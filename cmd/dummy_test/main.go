/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package main

import (
	"context"
	"os"

	"github.com/insolar/loaderbot"
)

func main() {
	target := os.Getenv("TARGET")
	r := loaderbot.NewRunner(&loaderbot.RunnerConfig{
		TargetUrl:       target,
		Name:            "dummy_test",
		SystemMode:      loaderbot.PrivateSystem,
		Attackers:       5000,
		AttackerTimeout: 25,
		StartRPS:        30000,
		// StepDurationSec: 10,
		// StepRPS:         1000,
		TestTimeSec:  3600,
		SuccessRatio: 0.95,
		Prometheus:   &loaderbot.Prometheus{Enable: true},
	}, &loaderbot.HTTPAttackerExample{}, nil)
	_, _ = r.Run(context.TODO())
}
