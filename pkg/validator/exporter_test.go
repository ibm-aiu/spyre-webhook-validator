/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator

import (
	spyrev1alpha1 "github.com/ibm-aiu/spyre-operator/api/v1alpha1"
)

func (v *ClusterPolicyHandler) GetSchedulerEnabled() bool {
	return v.schedulerEnabled.Load()
}

func (v *ClusterPolicyHandler) ValidateClusterPolicy(clusterPolicy spyrev1alpha1.SpyreClusterPolicy) error {
	return v.validate(clusterPolicy)
}

var IsSpyrePod = isSpyrePod
