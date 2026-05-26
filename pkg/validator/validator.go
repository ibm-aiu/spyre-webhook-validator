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
