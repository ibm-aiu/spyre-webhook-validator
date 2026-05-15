/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package main_test

import (
	"bytes"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	validator "github.com/ibm-aiu/spyre-webhook-validator"
)

var _ = Describe("Main", Label("main"), Ordered, func() {

	It("starts with info loglevel by default", func() {
		var logBuf bytes.Buffer
		GinkgoWriter.TeeTo(&logBuf)
		validator.PrepareLogger()
		Expect(logBuf.String()).Should(ContainSubstring("INFO"))
	})

	It("starts with debug loglevel when LOGLEVEL is set to \"debug\"", func() {
		Expect(os.Setenv("LOGLEVEL", "debug")).To(Succeed())
		var logBuf bytes.Buffer
		GinkgoWriter.TeeTo(&logBuf)
		validator.PrepareLogger()
		Expect(logBuf.String()).Should(ContainSubstring("\"loglevel\": \"debug\""))
	})
})
