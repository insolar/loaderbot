/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package main

import (
	"context"

	"github.com/insolar/loaderbot"
	"github.com/insolar/loaderbot/examples/attackers"
)

func main() {
	cfg := &loaderbot.RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "runner_1",
		SystemMode:      loaderbot.OpenWorldSystem,
		AttackerTimeout: 5,
		StartRPS:        5,
		StepDurationSec: 5,
		StepRPS:         1,
		TestTimeSec:     200,
	}
	lt := loaderbot.NewRunner(
		cfg,
		&attackers.DataAttackerExample{},
		loaderbot.NewSharedDataSlice([]interface{}{"data1", "data2", "data3"}),
	)
	_, _ = lt.Run(context.TODO())
}
