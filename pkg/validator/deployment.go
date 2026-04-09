/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator

import (
	"encoding/json"

	appsv1 "k8s.io/api/apps/v1"
)

type DeploymentValidator struct {
	*PodValidator
}

func NewDeploymentValidator(clusterPolicyHandler *ClusterPolicyHandler) *DeploymentValidator {
	return &DeploymentValidator{PodValidator: NewPodValidator(clusterPolicyHandler)}
}

func (v *DeploymentValidator) Validate(raw []byte) error {
	var deploy *appsv1.Deployment
	if err := json.Unmarshal(raw, &deploy); err != nil {
		return err
	}
	return v.PodValidator.ValidatePod(deploy.Spec.Template.Spec)
}
