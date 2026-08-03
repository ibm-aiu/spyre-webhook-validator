/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	spyrev1alpha1 "github.com/ibm-aiu/spyre-operator/api/v1alpha1"
	"github.com/ibm-aiu/spyre-webhook-validator/pkg/validator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	opNs = "spyre-operator"
)

var cfg *rest.Config
var testEnv *envtest.Environment
var k8sClient client.Client
var k8sClientset *kubernetes.Clientset

func resourceList(name corev1.ResourceName, quantity resource.Quantity) corev1.ResourceList {
	return map[corev1.ResourceName]resource.Quantity{
		name: quantity,
	}
}

func newPodValidator(state spyrev1alpha1.State, schedulerEnabled bool) *validator.PodValidator {
	if schedulerEnabled {
		GinkgoT().Setenv("EXTERNAL_DEVICE_RESERVATION_MODE", "1")
	} else {
		GinkgoT().Setenv("EXTERNAL_DEVICE_RESERVATION_MODE", "")
	}
	return validator.NewPodValidator(validator.NewClusterPolicyHandler())
}

func newJobValidator() *validator.JobValidator {
	GinkgoT().Setenv("EXTERNAL_DEVICE_RESERVATION_MODE", "1")
	return validator.NewJobValidator(validator.NewClusterPolicyHandler())
}

func newDeploymentValidator() *validator.DeploymentValidator {
	GinkgoT().Setenv("EXTERNAL_DEVICE_RESERVATION_MODE", "1")
	return validator.NewDeploymentValidator(validator.NewClusterPolicyHandler())
}

var _ = Describe("Pod", func() {

	oneQuant, err := resource.ParseQuantity("1")
	Expect(err).To(BeNil())

	Context("resource check", func() {

		Context("no-scheduler", func() {

			schedulerEnabled := false

			It("allows Pod without spyre_pf", func() {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "i",
					}},
				}
				err := v.ValidatePod(pSpec)
				Expect(err).To(BeNil())
			})

			It("allows Pod with one specific spyre_pf", func() {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "i",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{"ibm.com/spyre_pf_0000_1a_00.0": oneQuant},
						},
					}},
				}
				err := v.ValidatePod(pSpec)
				Expect(err).To(BeNil())
			})

			It("denies Pod with two specific spyre_pf", func() {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "i",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"ibm.com/spyre_pf_0000_1a_00.0": oneQuant,
								"ibm.com/spyre_pf_0000_3d_00.0": oneQuant},
						},
					}},
				}
				err := v.ValidatePod(pSpec)
				Expect(err).Should(Equal(validator.ErrMoreThanOneSpyreResource))
			})

			It("denies Pod with spyre_pf_tier0 and specific spyre_pf", func() {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "i",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"ibm.com/spyre_pf_tier0":        oneQuant,
								"ibm.com/spyre_pf_0000_1a_00.0": oneQuant},
						},
					}},
				}
				err := v.ValidatePod(pSpec)
				Expect(err).Should(Equal(validator.ErrMoreThanOneSpyreResource))
			})

			It("denies Pod with spyre_pf and spyre_pf_tier0", func() {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "i",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"ibm.com/spyre_pf":       oneQuant,
								"ibm.com/spyre_pf_tier0": oneQuant},
						},
					}},
				}
				err := v.ValidatePod(pSpec)
				Expect(err).Should(Equal(validator.ErrMoreThanOneSpyreResource))
			})

			DescribeTable("check invalid amount of tier resource", func(requests, limits corev1.ResourceList, expectedErr bool) {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "i",
						Resources: corev1.ResourceRequirements{
							Requests: requests,
							Limits:   limits,
						},
					}},
				}
				err := v.ValidatePod(pSpec)
				if expectedErr {
					Expect(err).Should(Equal(validator.ErrInvalidResourceAmount))
				} else {
					Expect(err).To(BeNil())
				}
			},
				Entry("empty request/limit", nil, nil, false),
				Entry("spyre_pf request/limit", resourceList("ibm.com/spyre_pf", oneQuant), resourceList("ibm.com/spyre_pf", oneQuant), false),
				Entry("one spyre_tier0 request/limit", resourceList("ibm.com/spyre_pf_tier0", oneQuant), resourceList("ibm.com/spyre_pf_tier0", oneQuant), false),
				Entry("even spyre_tier0 request/limit", resourceList("ibm.com/spyre_pf_tier0", resource.MustParse("2")), resourceList("ibm.com/spyre_pf_tier0", resource.MustParse("2")), false),
				Entry("odd spyre_tier0 request", resourceList("ibm.com/spyre_pf_tier0", resource.MustParse("3")), nil, true),
				Entry("odd spyre_tier0 limit", nil, resourceList("ibm.com/spyre_pf_tier0", resource.MustParse("3")), true),
				Entry("odd spyre_tier1 request", resourceList("ibm.com/spyre_pf_tier0", resource.MustParse("3")), nil, true),
				Entry("odd spyre_tier1 limit", nil, resourceList("ibm.com/spyre_pf_tier0", resource.MustParse("3")), true),
				Entry("odd spyre_tier2 request", resourceList("ibm.com/spyre_pf_tier0", resource.MustParse("3")), nil, true),
				Entry("odd spyre_tier2 limit", nil, resourceList("ibm.com/spyre_pf_tier0", resource.MustParse("3")), true),
			)
		})

		Context("with-scheduler", func() {

			schedulerEnabled := true

			It("denies Pod with two specific spyre_pf even if spyre-scheduler is specified", func() {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					SchedulerName: "spyre-scheduler",
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "i",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"ibm.com/spyre_pf_0000_1a_00.0": oneQuant,
								"ibm.com/spyre_pf_0000_3d_00.0": oneQuant},
						},
					}},
				}
				err := v.ValidatePod(pSpec)
				Expect(err).Should(Equal(validator.ErrMoreThanOneSpyreResource))
			})

			DescribeTable("deny incorrect scheduler name", func(resourceName string) {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					SchedulerName: "not-a-spyre-scheduler",
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "i",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceName(resourceName): oneQuant,
							},
						}},
					}}
				err := v.ValidatePod(pSpec)
				Expect(err).Should(Equal(validator.ErrNoSpyreScheduler))
			},
				Entry("spyre_pf", "ibm.com/spyre_pf"),
				Entry("spyre_tier0", "ibm.com/spyre_tier0"),
				Entry("spyre_pf_0000_1a_00", "ibm.com/spyre_pf_0000_1a_00.0"),
				Entry("spyre_vf", "ibm.com/spyre_vf"),
			)

			It("skips validate Pod without Spyre resources (allow)", func() {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "i",
					}},
				}
				err := v.ValidatePod(pSpec)
				Expect(err).To(BeNil())
			})
		})
	})

	Context("scheduler check", func() {

		Context("no-scheduler", func() {

			schedulerEnabled := false

			It("does not deny Pod with .spec.schedulerName != spyre-scheduler", func() {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					Containers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"ibm.com/spyre_pf": oneQuant,
							},
						},
					}},
					SchedulerName: "not-an-spyre-scheduler",
				}
				Expect(validator.IsSpyrePod(pSpec)).To(Equal(true))
				err := v.ValidatePod(pSpec)
				Expect(err).To(BeNil())
			})
		})

		Context("with-scheduler", func() {

			schedulerEnabled := true

			It("allows Pod with .spec.schedulerName = spyre-scheduler", func() {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					Containers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"ibm.com/spyre_pf": oneQuant,
							},
						},
					}},
					SchedulerName: "spyre-scheduler",
				}
				Expect(validator.IsSpyrePod(pSpec)).To(Equal(true))
				err := v.ValidatePod(pSpec)
				Expect(err).To(BeNil())
			})

			It("denies Pod with nodeName", func() {
				v := newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)
				pSpec := corev1.PodSpec{
					SchedulerName: "spyre-scheduler",
					NodeName:      "myNode",
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "i",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"ibm.com/spyre_pf": oneQuant,
							},
						},
					}},
				}
				err := v.ValidatePod(pSpec)
				Expect(err).Should(Equal(validator.ErrNodeNameWithScheduler))
			})
		})
	})

	Context("cluster policy check", func() {
		DescribeTable("state (scheduler enabled)", func(state spyrev1alpha1.State, schedulerEnabled bool, expectedAllowed bool) {
			v := newPodValidator(state, schedulerEnabled)
			pSpec := corev1.PodSpec{
				Containers: []corev1.Container{{
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"ibm.com/spyre_pf": oneQuant,
						},
					},
				}},
			}
			if schedulerEnabled {
				pSpec.SchedulerName = "spyre-scheduler"
			}
			Expect(validator.IsSpyrePod(pSpec)).To(Equal(true))
			err := v.ValidatePod(pSpec)
			if expectedAllowed {
				Expect(err).To(BeNil())
			} else {
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("cannot deploy any Spyre pods"))
			}
		},
			Entry("ready (true)", spyrev1alpha1.Ready, true, true),
			Entry("ready (false)", spyrev1alpha1.Ready, false, true),
			Entry("not ready (true)", spyrev1alpha1.NotReady, true, true),
			Entry("not ready (false)", spyrev1alpha1.NotReady, false, true),
		)
	})

	Context("envtest", Ordered, func() {
		schedulerEnabled := true
		var stopFunc func()
		ctx := context.Background()

		BeforeAll(func() {

			os.Setenv("OPERATOR_NAMESPACE", opNs)

			var err error

			testEnv = &envtest.Environment{
				ErrorIfCRDPathMissing: true,
				WebhookInstallOptions: envtest.WebhookInstallOptions{Paths: []string{
					filepath.Join("..", "..", "test", "manifest", "webhook-config")}},
			}

			// cfg is defined in this file globally.
			cfg, err = testEnv.Start()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
			k8sClientset = kubernetes.NewForConfigOrDie(cfg)
			Expect(k8sClientset).NotTo(BeNil())

			// build scheme
			err = scheme.AddToScheme(scheme.Scheme) // add default APIs
			Expect(err).NotTo(HaveOccurred())

			//+kubebuilder:scaffold:scheme

			k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient).NotTo(BeNil())
			createNamespace(opNs)

			// setup webhook
			webhookInstallOptions := &testEnv.WebhookInstallOptions
			mgr, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme:         scheme.Scheme,
				LeaderElection: false,
			})
			Expect(err).To(BeNil())

			hookServer := webhook.NewServer(webhook.Options{
				Port:    webhookInstallOptions.LocalServingPort,
				CertDir: webhookInstallOptions.LocalServingCertDir,
				Host:    webhookInstallOptions.LocalServingHost,
			})
			err = mgr.Add(hookServer)
			Expect(err).To(Succeed())

			hookServer.Register("/validate-pods", &webhook.Admission{
				Handler: validator.AdmissionHandler{Validator: newPodValidator(spyrev1alpha1.Ready, schedulerEnabled)}})
			hookServer.Register("/validate-jobs", &webhook.Admission{
				Handler: validator.AdmissionHandler{Validator: newJobValidator()}})
			hookServer.Register("/validate-deployments", &webhook.Admission{
				Handler: validator.AdmissionHandler{Validator: newDeploymentValidator()}})
			hookServer.Register("/validate-clusterpolicy", &webhook.Admission{
				Handler: validator.AdmissionHandler{Validator: validator.NewClusterPolicyHandler()}})

			ctx, cancel := context.WithCancel(ctx)
			stopFunc = cancel
			go func() {
				err := mgr.Start(ctx)
				if err != nil {
					panic(err)
				}
			}()

			// a namespace excluded in test/manifest/webhook-config/validatingwebhookconfig.yaml
			createNamespace("openshift-apiserver")

		})

		AfterAll(func() {
			By("tearing down the test environment")
			stopFunc()
			Eventually(func(g Gomega) {
				err := testEnv.Stop()
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(60 * time.Second).WithPolling(1000 * time.Millisecond).Should(Succeed())
		})

		DescribeTable("no scheduler", func(ns, schedName string, admissionErrorExpected bool) {
			ctx := context.Background()
			p := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      petname.Generate(2, "-"),
					Namespace: ns,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "c1",
						Image: "image",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"ibm.com/spyre_pf": oneQuant},
						},
					}},
					SchedulerName: schedName,
				},
			}
			err := k8sClient.Create(ctx, p)
			Expect(isAdmissionError(err)).Should(Equal(admissionErrorExpected))
		},
			Entry("default/no-sched triggers admissionError", "default", "", true),
			Entry("default/spyre-sched does not trigger admissionError", "default", "spyre-scheduler", false),
			Entry("spyre-operator/no-sched does not trigger admissionError", "spyre-operator", "", false),
			Entry("openshift-apiserver/no-sched does not trigger admissionError", "openshift-apiserver", "", false),
		)

	})

})

func isAdmissionError(err error) bool {
	if err == nil {
		return false
	}
	statusError, ok := err.(*errors.StatusError)
	if !ok {
		return false
	}
	msg := statusError.ErrStatus.Message
	if statusError.ErrStatus.Status == "Failure" &&
		strings.Contains(msg, "admission webhook") &&
		strings.Contains(msg, "denied the request") {
		return true
	}
	return false
}

func createNamespace(ns string) {
	ctx := context.Background()
	nsObj := &corev1.Namespace{}
	nsObj.Name = ns
	err := k8sClient.Create(ctx, nsObj)
	Expect(err).To(BeNil())
}
