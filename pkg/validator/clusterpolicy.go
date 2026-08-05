/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	spyrev1alpha1 "github.com/ibm-aiu/spyre-operator/api/v1alpha1"
)

const (
	ValidClusterPolicyName = "spyreclusterpolicy"
)

type ClusterPolicyHandler struct {
	schedulerEnabled bool
}

func NewClusterPolicyHandler() *ClusterPolicyHandler {
	return &ClusterPolicyHandler{
		schedulerEnabled: os.Getenv("EXTERNAL_DEVICE_RESERVATION_MODE") == "1",
	}
}

func (v *ClusterPolicyHandler) Validate(raw []byte) error {
	var clusterPolicy spyrev1alpha1.SpyreClusterPolicy
	if err := json.Unmarshal(raw, &clusterPolicy); err != nil {
		return err
	}
	return v.validate(clusterPolicy)
}

func (v *ClusterPolicyHandler) validate(clusterPolicy spyrev1alpha1.SpyreClusterPolicy) error {
	if err := v.validateSinglePolicy(clusterPolicy); err != nil {
		return fmt.Errorf("failed to validate cluster: %w", err)
	}
	if err := validateDevicePlugin(clusterPolicy.Spec.DevicePlugin); err != nil {
		return fmt.Errorf("failed to validate device plugin: %w", err)
	}
	schedulerEnabled := isSchedulerEnabled(clusterPolicy.Spec.ExperimentalMode)
	if err := validateScheduler(clusterPolicy.Spec.Scheduler, schedulerEnabled); err != nil {
		return fmt.Errorf("failed to validate scheduler: %w", err)
	}
	if err := validateExporter(clusterPolicy.Spec.MetricsExporter); err != nil {
		return fmt.Errorf("failed to validate exporter: %w", err)
	}
	if err := validateValidator(clusterPolicy.Spec.PodValidator); err != nil {
		return fmt.Errorf("failed to validate validator: %w", err)
	}
	return nil
}

func (v *ClusterPolicyHandler) validateSinglePolicy(clusterPolicy spyrev1alpha1.SpyreClusterPolicy) error {
	if clusterPolicy.Name != ValidClusterPolicyName {
		return fmt.Errorf("SpyreClusterPolicy must name %s to ensure a single cluster policy", ValidClusterPolicyName)
	}
	return nil
}

func validateDevicePlugin(devicePlugin spyrev1alpha1.DevicePluginSpec) error {
	err := validateDeployConfig(devicePlugin.DeploymentConfig)
	if err != nil {
		return WrapConfigErr(err)
	}
	return nil
}

func validateScheduler(scheduler spyrev1alpha1.SchedulerSpec, schedulerEnabled bool) error {
	if schedulerEnabled {
		err := validateDeployConfig(scheduler.DeploymentConfig)
		if err != nil {
			return WrapConfigErr(err)
		}
	}
	return nil
}

func isSchedulerEnabled(modes []spyrev1alpha1.SpyreClusterPolicyExperimentalMode) bool {
	for _, m := range modes {
		if m == spyrev1alpha1.ReservationMode {
			return true
		}
	}
	return false
}

func validateExporter(exporter spyrev1alpha1.MetricsExporterSpec) error {
	if exporter.Enabled {
		err := validateDeployConfig(exporter.DeploymentConfig)
		if err != nil {
			return WrapConfigErr(err)
		}
	}
	return nil
}

func validateValidator(validator spyrev1alpha1.PodValidatorSpec) error {
	if validator.Enabled {
		err := validateDeployConfig(validator.DeploymentConfig)
		if err != nil {
			return WrapConfigErr(err)
		}
	}
	return nil
}

func validateDeployConfig(config spyrev1alpha1.DeploymentConfig) error {
	if config.Image == "" {
		return errors.New("image not specified")
	}
	return nil
}
