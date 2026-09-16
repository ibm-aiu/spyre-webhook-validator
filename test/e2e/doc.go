/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

// Package e2e holds end-to-end tests that run against a live cluster with the
// webhook validator deployed. The actual specs are guarded by the `e2e` build
// tag (run them with `make e2e-test`); this file has no build tag so the
// package still builds under `make test` when the tag is absent.
package e2e
