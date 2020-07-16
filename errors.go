package loaderbot

import (
	"errors"
)

var (
	errAttackDoTimedOut = errors.New("attack Do(ctx) timeout")
	errAttackerSetup    = errors.New("error when setup attacker")
)
