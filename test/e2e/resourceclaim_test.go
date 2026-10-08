//go:build e2e

/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package e2e_test

import (
	spyrev1alpha1 "github.com/ibm-aiu/spyre-operator/api/v1alpha1"
	"github.com/ibm-aiu/spyre-webhook-validator/pkg/validator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	resourcev1 "k8s.io/api/resource/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// applyPolicy creates or updates the singleton SpyreClusterPolicy so that the
// webhook caches the requested DRA driver and VF mode. The policy must be named
// "spyreclusterpolicy" and carry a device-plugin image, otherwise
// /validate-clusterpolicy rejects it and the cache is not updated.
func applyPolicy(draDriver, vfMode bool) {
	policy := &spyrev1alpha1.SpyreClusterPolicy{}
	err := k8s.Get(ctx, client.ObjectKey{Name: policyName}, policy)
	if err != nil && !apierrors.IsNotFound(err) {
		Expect(err).NotTo(HaveOccurred())
	}

	policy.Name = policyName
	policy.Spec.DevicePlugin.DRADriver = draDriver
	policy.Spec.DevicePlugin.Image = "spyre-device-plugin"
	policy.Spec.CardManagement.Enabled = vfMode

	if policy.ResourceVersion == "" {
		Expect(k8s.Create(ctx, policy)).To(Succeed())
	} else {
		Expect(k8s.Update(ctx, policy)).To(Succeed())
	}
}

// tryCreateClaim attempts to create a ResourceClaim requesting deviceClass in
// the spyre-apps namespace and returns the admission error (nil when allowed).
// Successfully created claims are cleaned up at the end of the spec.
func tryCreateClaim(name, deviceClass string) error {
	claim := &resourcev1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: appsNamespace},
		Spec: resourcev1.ResourceClaimSpec{
			Devices: resourcev1.DeviceClaim{
				Requests: []resourcev1.DeviceRequest{{
					Name:    "req0",
					Exactly: &resourcev1.ExactDeviceRequest{DeviceClassName: deviceClass},
				}},
			},
		},
	}
	err := k8s.Create(ctx, claim)
	if err == nil {
		DeferCleanup(func() {
			_ = k8s.Delete(ctx, claim)
		})
	}
	return err
}

var _ = Describe("ResourceClaim webhook (spyre-apps namespace)", func() {

	Context("DRA driver ON, VF mode ON", Ordered, func() {
		BeforeAll(func() {
			applyPolicy(true, true)
		})

		It("allows spyre-standard-vf", func() {
			Expect(tryCreateClaim("e2e-vfon-standard-vf", validator.DeviceClassSpyreStandardVF)).To(Succeed())
		})

		It("denies spyre-pf", func() {
			err := tryCreateClaim("e2e-vfon-pf", validator.DeviceClassSpyrePF)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(validator.ErrSpyrePFRestrictedToOperatorNamespace.Error()))
		})

		It("denies spyre-privileged-vf", func() {
			err := tryCreateClaim("e2e-vfon-privileged-vf", validator.DeviceClassSpyrePrivilegedVF)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(validator.ErrPrivilegedVFRestrictedToOperatorNamespace.Error()))
		})
	})

	Context("DRA driver ON, VF mode OFF", Ordered, func() {
		BeforeAll(func() {
			applyPolicy(true, false)
		})

		It("allows spyre-pf", func() {
			Expect(tryCreateClaim("e2e-vfoff-pf", validator.DeviceClassSpyrePF)).To(Succeed())
		})

		It("denies spyre-privileged-vf", func() {
			err := tryCreateClaim("e2e-vfoff-privileged-vf", validator.DeviceClassSpyrePrivilegedVF)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(validator.ErrPrivilegedVFNotAllowedInNonVFMode.Error()))
		})

		It("denies spyre-standard-vf", func() {
			err := tryCreateClaim("e2e-vfoff-standard-vf", validator.DeviceClassSpyreStandardVF)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(validator.ErrStandardVFNotAllowedInNonVFMode.Error()))
		})
	})

	Context("DRA driver OFF", Ordered, func() {
		// With the classic device plugin (draDriver false), the ResourceClaim
		// webhook imposes no restriction even for a class that would be denied
		// under DRA (spyre-pf in an app namespace with VF mode on).
		BeforeAll(func() {
			applyPolicy(false, true)
		})

		It("allows spyre-pf", func() {
			Expect(tryCreateClaim("e2e-nodra-pf", validator.DeviceClassSpyrePF)).To(Succeed())
		})

		It("allows spyre-privileged-vf", func() {
			Expect(tryCreateClaim("e2e-nodra-privileged-vf", validator.DeviceClassSpyrePrivilegedVF)).To(Succeed())
		})
	})
})
