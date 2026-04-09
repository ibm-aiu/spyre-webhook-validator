/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator_test

import (
	"context"
	"encoding/json"

	spyrev1alpha1 "github.com/ibm-aiu/spyre-operator/api/v1alpha1"
	"github.com/ibm-aiu/spyre-webhook-validator/pkg/validator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	validConfig   = spyrev1alpha1.DeploymentConfig{Image: "some-image"}
	invalidConfig = spyrev1alpha1.DeploymentConfig{Image: ""}
)

func genValidClusterPolicy() spyrev1alpha1.SpyreClusterPolicy {
	clusterPolicy := spyrev1alpha1.SpyreClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: validator.ValidClusterPolicyName,
		},
		Spec: spyrev1alpha1.SpyreClusterPolicySpec{
			DevicePlugin: spyrev1alpha1.DevicePluginSpec{
				DeploymentConfig: validConfig,
			},
			Scheduler: spyrev1alpha1.SchedulerSpec{
				DeploymentConfig: validConfig,
			},
			MetricsExporter: spyrev1alpha1.MetricsExporterSpec{
				DeploymentConfig: validConfig,
			},
			PodValidator: spyrev1alpha1.PodValidatorSpec{
				DeploymentConfig: validConfig,
			},
			ExperimentalMode: []spyrev1alpha1.SpyreClusterPolicyExperimentalMode{
				spyrev1alpha1.ReservationMode,
			},
		},
	}
	return clusterPolicy
}

var _ = Describe("SpyreClusterPolicy", func() {
	DescribeTable("validate configs", func(devicePluginConfig,
		schedulerConfig, exporterConfig, validatorConfig spyrev1alpha1.DeploymentConfig,
		schedulerEnabled, exporterEnabled, validatorEnabled bool, expectedErrSubstring string) {
		clusterPolicy := spyrev1alpha1.SpyreClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: validator.ValidClusterPolicyName,
			},
			Spec: spyrev1alpha1.SpyreClusterPolicySpec{
				DevicePlugin: spyrev1alpha1.DevicePluginSpec{
					DeploymentConfig: devicePluginConfig,
				},
				Scheduler: spyrev1alpha1.SchedulerSpec{
					DeploymentConfig: schedulerConfig,
				},
				MetricsExporter: spyrev1alpha1.MetricsExporterSpec{
					DeploymentConfig: exporterConfig,
				},
				PodValidator: spyrev1alpha1.PodValidatorSpec{
					DeploymentConfig: validatorConfig,
				},
			},
		}
		if schedulerEnabled {
			clusterPolicy.Spec.ExperimentalMode = append(clusterPolicy.Spec.ExperimentalMode, spyrev1alpha1.ReservationMode)
		}
		if exporterEnabled {
			clusterPolicy.Spec.MetricsExporter.Enabled = true
		}
		if validatorEnabled {
			clusterPolicy.Spec.PodValidator.Enabled = true
		}
		v := validator.NewClusterPolicyHandler()
		err := v.ValidateClusterPolicy(clusterPolicy)
		if expectedErrSubstring != "" {
			Expect(err.Error()).To(ContainSubstring(expectedErrSubstring))
		} else {
			Expect(err).To(BeNil())
		}
	},
		Entry("all valid", validConfig, validConfig, validConfig, validConfig, false, false, false, ""),
		Entry("invalid device plugin", invalidConfig, validConfig, validConfig, validConfig, false, false, false, "device plugin"),
		Entry("invalid scheduler", validConfig, invalidConfig, validConfig, validConfig, true, false, false, "scheduler"),
		Entry("invalid scheduler (disabled)", validConfig, invalidConfig, validConfig, validConfig, false, false, false, ""),
		Entry("invalid exporter", validConfig, validConfig, invalidConfig, validConfig, false, true, false, "exporter"),
		Entry("invalid exporter (disabled)", validConfig, validConfig, invalidConfig, validConfig, false, false, false, ""),
		Entry("invalid validator", validConfig, validConfig, validConfig, invalidConfig, false, false, true, "validator"),
		Entry("invalid validator (disabled)", validConfig, validConfig, validConfig, invalidConfig, false, false, false, ""),
	)

	Context("single policy", func() {
		It("validate name", func() {
			v := validator.NewClusterPolicyHandler()
			clusterPolicy := genValidClusterPolicy()
			err := v.ValidateClusterPolicy(clusterPolicy)
			Expect(err).To(BeNil())
			By("checking value set")
			Expect(v.GetSchedulerEnabled()).To(BeTrue())
			By("testing same name")
			err = v.ValidateClusterPolicy(clusterPolicy)
			Expect(err).To(BeNil())
			By("changing name")
			clusterPolicy.Name = "new-cluster"
			err = v.ValidateClusterPolicy(clusterPolicy)
			Expect(err.Error()).To(ContainSubstring(validator.ValidClusterPolicyName))
		})

		It("allow creating, deleting and recreating request with valid name", func() {
			ctx := context.Background()
			clusterPolicy := genValidClusterPolicy()
			handler := validator.AdmissionHandler{Validator: validator.NewClusterPolicyHandler()}
			raw, err := json.Marshal(clusterPolicy)
			Expect(err).To(BeNil())
			updateRequest := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: raw,
					},
					Operation: admissionv1.Update,
				},
			}
			deleteRequest := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: raw,
					},
					Operation: admissionv1.Delete,
				},
			}
			By("creating")
			response := handler.Handle(ctx, updateRequest)
			Expect(response.Allowed).To(BeTrue())
			By("deleting cluster policy")
			response = handler.Handle(ctx, deleteRequest)
			Expect(response.Allowed).To(BeTrue())
			By("recreating")
			response = handler.Handle(ctx, updateRequest)
			Expect(response.Allowed).To(BeTrue())
		})

		It("deny creating request with invalid name", func() {
			ctx := context.Background()
			clusterPolicy := genValidClusterPolicy()
			clusterPolicy.Name = "invalid-name"
			handler := validator.AdmissionHandler{Validator: validator.NewClusterPolicyHandler()}
			raw, err := json.Marshal(clusterPolicy)
			Expect(err).To(BeNil())
			updateRequest := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: raw,
					},
					Operation: admissionv1.Update,
				},
			}
			response := handler.Handle(ctx, updateRequest)
			Expect(response.Allowed).To(BeFalse())
		})
	})

})
