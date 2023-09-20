package loaderbot

import (
	"errors"
)

var (
	errAttackDoTimedOut = "attack Do(ctx) timeout"
	errAttackerSetup    = errors.New("error when setup attacker")
	errNodeIsBusy       = "client %s is busy, skipping test"
)
