/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	SpyrePrefix        = "ibm.com/spyre"
	SpyreSchedulerName = "spyre-scheduler"
	SpyreAnnotationKey = "k8s.ibm.pci.io/spyre"
)

type PodValidator struct {
	*ClusterPolicyHandler
}

// NewPodValidator returns a new PodValidator instance.
func NewPodValidator(clusterPolicyHandler *ClusterPolicyHandler) *PodValidator {
	return &PodValidator{ClusterPolicyHandler: clusterPolicyHandler}
}

func (v *PodValidator) Validate(raw []byte) error {
	var pod *corev1.Pod
	if err := json.Unmarshal(raw, &pod); err != nil {
		return err
	}
	return v.ValidatePod(pod.Spec)
}

func (v *PodValidator) ValidatePod(spec corev1.PodSpec) error {
	if !isSpyrePod(spec) {
		return nil
	}

	if v.ClusterPolicyHandler.schedulerEnabled.Load() {
		if spec.SchedulerName != SpyreSchedulerName {
			return NoSpyreSchedulerErr
		}
		if spec.NodeName != "" {
			return NodeNameWithSchedulerErr
		}
	}

	for _, container := range spec.Containers {
		if container.Resources.Requests != nil {
			if hasMoreThanOneSpyreResources(container.Resources.Requests) {
				return MoreThanOneSpyreResourcesErr
			}
			if !spyreTierResourceIsEvenNumberOrOne(container.Resources.Requests) {
				return InvalidResourceAmountErr
			}
		}
		if container.Resources.Limits != nil {
			if hasMoreThanOneSpyreResources(container.Resources.Limits) {
				return MoreThanOneSpyreResourcesErr
			}
			if !spyreTierResourceIsEvenNumberOrOne(container.Resources.Limits) {
				return InvalidResourceAmountErr
			}
		}
	}
	return nil
}

// isSpyrePod returns true if the given Pod limits or requests Spyre.
func isSpyrePod(spec corev1.PodSpec) bool {
	for _, c := range spec.Containers {
		for k := range c.Resources.Requests {
			if strings.HasPrefix(string(k), SpyrePrefix) {
				return true
			}
		}
		for k := range c.Resources.Limits {
			if strings.HasPrefix(string(k), SpyrePrefix) {
				return true
			}
		}
	}
	return false
}

func hasMoreThanOneSpyreResources(resourceList corev1.ResourceList) bool {
	spyreRequested := false
	for name := range resourceList {
		if strings.HasPrefix(name.String(), SpyrePrefix) {
			if spyreRequested {
				return true
			}
			spyreRequested = true
		}
	}
	return false
}

func spyreTierResourceIsEvenNumberOrOne(resourceList corev1.ResourceList) bool {
	for name, quantity := range resourceList {
		if isTierResourceRequest(name.String()) {
			amount, success := quantity.AsInt64()
			if !success {
				return false
			}
			if amount != 1 && amount%2 == 1 {
				fmt.Printf("invalid amount %d", amount)
				return false
			}
		}
	}
	return true
}

func isTierResourceRequest(name string) bool {
	return strings.HasPrefix(name, SpyrePrefix) && strings.Contains(name, "tier")
}
