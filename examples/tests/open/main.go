package main

import (
	"context"
	"fmt"

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
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         10,
		TestTimeSec:     200,
	}
	lt := loaderbot.NewRunner(cfg, &attackers.AttackerExample{}, nil)
	maxRPS, _ := lt.Run(context.TODO())
	fmt.Printf("max rps: %.2f", maxRPS)
}
