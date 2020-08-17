/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"errors"
)

var (
	errAttackDoTimedOut = "attack Do(ctx) timeout"
	errAttackerSetup    = errors.New("error when setup attacker")
	errNodeIsBusy       = "client %s is busy, skipping test"
)
