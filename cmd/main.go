package main

import (
	"fmt"

	"github.com/insolar/loaderbot"
)

func main() {
	cfg := &loaderbot.RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "abc",
		Attackers:       10,
		AttackerTimeout: 5,
		StartRPS:        100,
		StepDurationSec: 10,
		StepRPS:         300,
		TestTimeSec:     200,
	}
	lt := loaderbot.NewRunner(cfg, &loaderbot.HTTPAttackerExample{}, nil)
	maxRPS, _ := lt.Run()
	fmt.Printf("max rps: %.2f", maxRPS)
}
