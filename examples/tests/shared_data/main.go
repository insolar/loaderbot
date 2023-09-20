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
		SystemMode:      loaderbot.BoundRPS,
		Attackers:       10,
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
