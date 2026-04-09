/*
 * +-------------------------------------------------------------------+
 * | Copyright (c) 2025, 2026 IBM Corp.                                |
 * | SPDX-License-Identifier: Apache-2.0                               |
 * +-------------------------------------------------------------------+
 */

package validator

import (
	"errors"
	"fmt"
)

var (
	MoreThanOneSpyreResourcesErr = errors.New("A pod cannot request or limit Spyre devices from more than one resource pool.")
	NoSpyreSchedulerErr          = errors.New("A pod must use \"schedulerName: spyre-scheduler\".")
	InvalidResourceAmountErr     = errors.New("A pod cannot request or limit Spyre devices for tier0, tier1, and tier2 with an odd number except 1.")             //nolint:lll
	NodeNameWithSchedulerErr     = errors.New("A pod must not use \".spec.nodeName\" with \"schedulerName: spyre-scheduler\"; use '.spec.nodeSelector' instead.") //nolint:lll
)

func WrapConfigErr(err error) error {
	return fmt.Errorf("failed to validate config: %w", err)
}
