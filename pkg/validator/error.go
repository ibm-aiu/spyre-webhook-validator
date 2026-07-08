/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator

import (
	"errors"
	"fmt"
)

var (
	ErrMoreThanOneSpyreResource = errors.New("a pod cannot request or limit Spyre devices from more than one resource pool")
	ErrNoSpyreScheduler         = errors.New("a pod must use \"schedulerName: spyre-scheduler\"")
	ErrInvalidResourceAmount    = errors.New("a pod cannot request or limit Spyre devices for tier0, tier1, and tier2 with an odd number except 1")             //nolint:lll
	ErrNodeNameWithScheduler    = errors.New("a pod must not use \".spec.nodeName\" with \"schedulerName: spyre-scheduler\"; use '.spec.nodeSelector' instead") //nolint:lll

	ErrSpyrePFRestrictedToOperatorNamespace      = errors.New("spyre-pf DeviceClass is restricted to the spyre-operator namespace in VF mode")
	ErrPrivilegedVFRestrictedToOperatorNamespace = errors.New("spyre-privileged-vf DeviceClass is restricted to the spyre-operator namespace")
	ErrPrivilegedVFNotAllowedInNonVFMode         = errors.New("spyre-privileged-vf DeviceClass is not allowed when VF mode is disabled")
	ErrStandardVFNotAllowedInNonVFMode           = errors.New("spyre-standard-vf DeviceClass is not allowed when VF mode is disabled")
)

func WrapConfigErr(err error) error {
	return fmt.Errorf("failed to validate config: %w", err)
}
