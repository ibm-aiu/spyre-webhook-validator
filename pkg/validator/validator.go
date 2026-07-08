/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator

import (
	"context"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Validator interface {
	Validate(raw []byte) error
}

type AdmissionHandler struct {
	Validator
}

func (h AdmissionHandler) Handle(ctx context.Context, request admission.Request) admission.Response {
	if err := h.Validate(request.Object.Raw); err != nil {
		return admission.Errored(http.StatusForbidden, err)
	}
	return admission.Allowed("")
}

// ResourceClaimAdmissionHandler is a webhook handler for ResourceClaim resources.
// Unlike AdmissionHandler it forwards the request namespace to the validator.
type ResourceClaimAdmissionHandler struct {
	Validator *ResourceClaimValidator
}

func (h ResourceClaimAdmissionHandler) Handle(ctx context.Context, request admission.Request) admission.Response {
	if err := h.Validator.Validate(request.Namespace, request.Object.Raw); err != nil {
		return admission.Errored(http.StatusForbidden, err)
	}
	return admission.Allowed("")
}
