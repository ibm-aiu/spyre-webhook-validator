/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator

import (
	"encoding/json"

	resourcev1 "k8s.io/api/resource/v1"
)

const (
	DeviceClassSpyrePF           = "spyre-pf"
	DeviceClassSpyrePrivilegedVF = "spyre-privileged-vf"
	DeviceClassSpyreStandardVF   = "spyre-standard-vf"
)

type accessLevel int

const (
	accessOpen         accessLevel = iota // claimable by any namespace
	accessOperatorOnly                    // claimable only by the operator namespace
	accessDenied                          // claimable by no one
)

type deviceClassRule struct {
	vfOnAccess  accessLevel
	vfOffAccess accessLevel
}

var deviceClassRules = map[string]deviceClassRule{
	DeviceClassSpyrePF:           {vfOnAccess: accessOperatorOnly, vfOffAccess: accessOpen},
	DeviceClassSpyrePrivilegedVF: {vfOnAccess: accessOperatorOnly, vfOffAccess: accessDenied},
	DeviceClassSpyreStandardVF:   {vfOnAccess: accessOpen, vfOffAccess: accessDenied},
}

// operatorOnlyErrors maps DeviceClass name to the error returned when
// accessOperatorOnly is selected but the namespace doesn't match.
var operatorOnlyErrors = map[string]error{
	DeviceClassSpyrePF:           ErrSpyrePFRestrictedToOperatorNamespace,
	DeviceClassSpyrePrivilegedVF: ErrPrivilegedVFRestrictedToOperatorNamespace,
}

// deniedErrors maps DeviceClass name to the error returned when accessDenied is selected.
var deniedErrors = map[string]error{
	DeviceClassSpyrePrivilegedVF: ErrPrivilegedVFNotAllowedInNonVFMode,
	DeviceClassSpyreStandardVF:   ErrStandardVFNotAllowedInNonVFMode,
}

// ResourceClaimValidator validates ResourceClaim resources.
type ResourceClaimValidator struct {
	clusterPolicyHandler *ClusterPolicyHandler
}

// NewResourceClaimValidator creates a new ResourceClaimValidator.
func NewResourceClaimValidator(cph *ClusterPolicyHandler) *ResourceClaimValidator {
	return &ResourceClaimValidator{clusterPolicyHandler: cph}
}

// Validate decodes the raw ResourceClaim and enforces DeviceClass access rules.
func (v *ResourceClaimValidator) Validate(namespace string, raw []byte) error {
	var claim resourcev1.ResourceClaim
	if err := json.Unmarshal(raw, &claim); err != nil {
		return err
	}

	vfMode := v.clusterPolicyHandler.vfModeEnabled.Load()
	operatorNS, _ := v.clusterPolicyHandler.operatorNamespace.Load().(string)

	for _, name := range collectDeviceClassNames(claim) {
		if err := validateDeviceClassName(name, namespace, vfMode, operatorNS); err != nil {
			return err
		}
	}
	return nil
}

// collectDeviceClassNames returns all DeviceClass names referenced by a ResourceClaim.
func collectDeviceClassNames(claim resourcev1.ResourceClaim) []string {
	var names []string
	for _, req := range claim.Spec.Devices.Requests {
		if req.Exactly != nil {
			names = append(names, req.Exactly.DeviceClassName)
		}
		for _, sub := range req.FirstAvailable {
			names = append(names, sub.DeviceClassName)
		}
	}
	return names
}

// validateDeviceClassName checks a single DeviceClass name against the access rules.
func validateDeviceClassName(name, namespace string, vfMode bool, operatorNS string) error {
	rule, ok := deviceClassRules[name]
	if !ok {
		return nil
	}

	var level accessLevel
	if vfMode {
		level = rule.vfOnAccess
	} else {
		level = rule.vfOffAccess
	}

	switch level {
	case accessOpen:
		return nil
	case accessDenied:
		return deniedErrors[name]
	case accessOperatorOnly:
		if namespace == operatorNS {
			return nil
		}
		return operatorOnlyErrors[name]
	}
	return nil
}
