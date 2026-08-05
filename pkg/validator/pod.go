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

	if err := v.validateScheduler(spec); err != nil {
		return err
	}

	allContainers := make([]corev1.Container, len(spec.Containers)+len(spec.InitContainers))
	copy(allContainers, spec.Containers)
	copy(allContainers[len(spec.Containers):], spec.InitContainers)
	return v.validateContainerResources(allContainers)
}

// validateScheduler checks if the scheduler configuration is valid for Spyre pods
func (v *PodValidator) validateScheduler(spec corev1.PodSpec) error {
	if !v.schedulerEnabled {
		return nil
	}

	if spec.SchedulerName != SpyreSchedulerName {
		return ErrNoSpyreScheduler
	}

	if spec.NodeName != "" {
		return ErrNodeNameWithScheduler
	}

	return nil
}

// validateContainerResources validates resource requests and limits for all containers
func (v *PodValidator) validateContainerResources(containers []corev1.Container) error {
	for _, container := range containers {
		if err := validateResourceList(container.Resources.Requests); err != nil {
			return err
		}
		if err := validateResourceList(container.Resources.Limits); err != nil {
			return err
		}
	}
	return nil
}

// validateResourceList validates a single resource list (requests or limits)
func validateResourceList(resourceList corev1.ResourceList) error {
	if resourceList == nil {
		return nil
	}

	if hasMoreThanOneSpyreResources(resourceList) {
		return ErrMoreThanOneSpyreResource
	}

	if !spyreTierResourceIsEvenNumberOrOne(resourceList) {
		return ErrInvalidResourceAmount
	}

	return nil
}

// isSpyrePod returns true if the given Pod limits or requests Spyre in any container,
// including init containers.
func isSpyrePod(spec corev1.PodSpec) bool {
	allContainers := make([]corev1.Container, len(spec.Containers)+len(spec.InitContainers))
	copy(allContainers, spec.Containers)
	copy(allContainers[len(spec.Containers):], spec.InitContainers)
	for _, c := range allContainers {
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
