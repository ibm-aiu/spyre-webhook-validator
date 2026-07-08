/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator_test

import (
	"encoding/json"
	"fmt"

	"github.com/ibm-aiu/spyre-webhook-validator/pkg/validator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	operatorNamespace = "spyre-operator"
	generalNamespace  = "default"
)

// newResourceClaimValidator returns a ResourceClaimValidator with the given VF mode and operator namespace.
func newResourceClaimValidator(vfMode bool) *validator.ResourceClaimValidator {
	cph := validator.NewClusterPolicyHandler()
	cph.SetVFModeEnabled(vfMode)
	cph.SetOperatorNamespace(operatorNamespace)
	return validator.NewResourceClaimValidator(cph)
}

// buildClaim builds a ResourceClaim JSON with a single exactly request.
func buildClaim(deviceClassName string) []byte {
	claim := resourcev1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "test-claim"},
		Spec: resourcev1.ResourceClaimSpec{
			Devices: resourcev1.DeviceClaim{
				Requests: []resourcev1.DeviceRequest{
					{
						Name: "req0",
						Exactly: &resourcev1.ExactDeviceRequest{
							DeviceClassName: deviceClassName,
						},
					},
				},
			},
		},
	}
	raw, err := json.Marshal(claim)
	Expect(err).To(BeNil())
	return raw
}

// buildClaimFirstAvailable builds a ResourceClaim JSON using firstAvailable.
func buildClaimFirstAvailable(deviceClassNames ...string) []byte {
	subs := make([]resourcev1.DeviceSubRequest, 0, len(deviceClassNames))
	for i, name := range deviceClassNames {
		subs = append(subs, resourcev1.DeviceSubRequest{
			Name:            fmt.Sprintf("sub%d", i),
			DeviceClassName: name,
		})
	}
	claim := resourcev1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "test-claim"},
		Spec: resourcev1.ResourceClaimSpec{
			Devices: resourcev1.DeviceClaim{
				Requests: []resourcev1.DeviceRequest{
					{
						Name:           "req0",
						FirstAvailable: subs,
					},
				},
			},
		},
	}
	raw, err := json.Marshal(claim)
	Expect(err).To(BeNil())
	return raw
}

var _ = Describe("ResourceClaim", func() {

	Context("VF mode ON", func() {

		v := newResourceClaimValidator(true)

		Context("operator namespace", func() {
			It("allows spyre-pf", func() {
				Expect(v.Validate(operatorNamespace, buildClaim(validator.DeviceClassSpyrePF))).To(BeNil())
			})
			It("allows spyre-privileged-vf", func() {
				Expect(v.Validate(operatorNamespace, buildClaim(validator.DeviceClassSpyrePrivilegedVF))).To(BeNil())
			})
			It("allows spyre-standard-vf", func() {
				Expect(v.Validate(operatorNamespace, buildClaim(validator.DeviceClassSpyreStandardVF))).To(BeNil())
			})
		})

		Context("general namespace", func() {
			It("denies spyre-pf", func() {
				Expect(v.Validate(generalNamespace, buildClaim(validator.DeviceClassSpyrePF))).
					To(MatchError(validator.ErrSpyrePFRestrictedToOperatorNamespace))
			})
			It("denies spyre-privileged-vf", func() {
				Expect(v.Validate(generalNamespace, buildClaim(validator.DeviceClassSpyrePrivilegedVF))).
					To(MatchError(validator.ErrPrivilegedVFRestrictedToOperatorNamespace))
			})
			It("allows spyre-standard-vf", func() {
				Expect(v.Validate(generalNamespace, buildClaim(validator.DeviceClassSpyreStandardVF))).To(BeNil())
			})
		})
	})

	Context("VF mode OFF", func() {

		v := newResourceClaimValidator(false)

		It("allows spyre-pf in any namespace", func() {
			Expect(v.Validate(generalNamespace, buildClaim(validator.DeviceClassSpyrePF))).To(BeNil())
		})
		It("denies spyre-privileged-vf", func() {
			Expect(v.Validate(generalNamespace, buildClaim(validator.DeviceClassSpyrePrivilegedVF))).
				To(MatchError(validator.ErrPrivilegedVFNotAllowedInNonVFMode))
		})
		It("denies spyre-standard-vf", func() {
			Expect(v.Validate(generalNamespace, buildClaim(validator.DeviceClassSpyreStandardVF))).
				To(MatchError(validator.ErrStandardVFNotAllowedInNonVFMode))
		})
	})

	Context("firstAvailable", func() {
		It("denies when a restricted class appears in firstAvailable (VF mode ON, general ns)", func() {
			v := newResourceClaimValidator(true)
			raw := buildClaimFirstAvailable(validator.DeviceClassSpyrePF, validator.DeviceClassSpyreStandardVF)
			Expect(v.Validate(generalNamespace, raw)).
				To(MatchError(validator.ErrSpyrePFRestrictedToOperatorNamespace))
		})
		It("allows when all classes in firstAvailable are open (VF mode ON, operator ns)", func() {
			v := newResourceClaimValidator(true)
			raw := buildClaimFirstAvailable(validator.DeviceClassSpyrePF, validator.DeviceClassSpyreStandardVF)
			Expect(v.Validate(operatorNamespace, raw)).To(BeNil())
		})
	})

	Context("non-Spyre DeviceClass", func() {
		It("is always allowed regardless of VF mode", func() {
			vOn := newResourceClaimValidator(true)
			vOff := newResourceClaimValidator(false)
			raw := buildClaim("some-other-device-class")
			Expect(vOn.Validate(generalNamespace, raw)).To(BeNil())
			Expect(vOff.Validate(generalNamespace, raw)).To(BeNil())
		})
	})
})
