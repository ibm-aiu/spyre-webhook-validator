//go:build e2e

/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package e2e_test

import (
	"context"
	"testing"

	spyrev1alpha1 "github.com/ibm-aiu/spyre-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	appsNamespace = "spyre-apps"
	policyName    = "spyreclusterpolicy"
)

var (
	ctx context.Context
	k8s client.Client
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ResourceClaim Webhook E2E Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()

	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
	Expect(resourcev1.AddToScheme(scheme)).To(Succeed())
	Expect(spyrev1alpha1.AddToScheme(scheme)).To(Succeed())

	cfg, err := config.GetConfig()
	Expect(err).NotTo(HaveOccurred(), "a reachable cluster is required (set KUBECONFIG / oc login)")

	k8s, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	// Ensure the test namespace exists (it is governed by the webhook).
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: appsNamespace}}
	if err := k8s.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).NotTo(HaveOccurred())
	}
})
