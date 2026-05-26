/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator

import (
	"encoding/json"

	batchv1 "k8s.io/api/batch/v1"
)

type JobValidator struct {
	*PodValidator
}

func NewJobValidator(clusterPolicyHandler *ClusterPolicyHandler) *JobValidator {
	return &JobValidator{PodValidator: NewPodValidator(clusterPolicyHandler)}
}

func (v *JobValidator) Validate(raw []byte) error {
	var job *batchv1.Job
	if err := json.Unmarshal(raw, &job); err != nil {
		return err
	}
	return v.ValidatePod(job.Spec.Template.Spec)
}
