/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"log"
	"os"
)

type SystemMode int

const (
	PrivateSystem SystemMode = iota
	OpenWorldSystem
)

// RunnerConfig runner configuration
type RunnerConfig struct {
	// TargetUrl target base url
	TargetUrl string
	// Name of a runner instance
	Name string
	// SystemMode PrivateSystem | OpenWorldSystem
	// PrivateSystem:
	// if application under test is a private system sync runner attackers will wait for response
	// in case your system is private and you know how many sync clients can act
	// OpenWorldSystem:
	// if application under test is an open world system async runner attackers will fire requests without waiting
	// it creates some inaccuracy in results, so you can check latencies using service metrics to be precise,
	// but the test will be more realistic from clients point of view
	SystemMode SystemMode
	// Attackers constant amount of attackers,
	// if SystemMode is "OpenWorldSystem", attackers will be spawn on demand to meet rps
	Attackers int
	// AttackerTimeout timeout of attacker
	AttackerTimeout int
	// StartRPS initial requests per seconds rate
	StartRPS int
	// StepDurationSec duration of step in which rps is increased by StepRPS
	StepDurationSec int
	// StepRPS amount of requests per second which will be added in next step,
	// if StepRPS = 0 rate is constant, default StepDurationSec is 30 sec is applied,
	// just to keep 30s aggregation metrics
	StepRPS int
	// TestTimeSec test timeout
	TestTimeSec int
	// WaitBeforeSec time to wait before start in case we didn't know start criteria
	WaitBeforeSec int
	// Dumptransport dump http requests to stdout
	DumpTransport bool
	// GoroutinesDump dumps goroutines stack for debug purposes
	GoroutinesDump bool
	// FailOnFirstError fails test on first error
	FailOnFirstError bool
	// LogLevel debug|info, etc.
	LogLevel string
	// LogEncoding json|console
	LogEncoding string
}

func (c *RunnerConfig) Validate() {
	errors := c.validate()
	if len(errors) > 0 {
		for _, e := range errors {
			log.Print(e)
		}
		os.Exit(1)
	}
}

func (c *RunnerConfig) DefaultCfgValues() {
	if c.SystemMode == OpenWorldSystem {
		// attacker will spawn goroutines for requests anyway, in this mode we are non-blocking
		c.Attackers = 1
	}
	// constant load
	if c.StepRPS == 0 {
		c.StepDurationSec = 10
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.LogEncoding == "" {
		c.LogEncoding = "console"
	}
}

// Validate checks all settings and returns a list of strings with problems.
func (c RunnerConfig) validate() (list []string) {
	if c.Attackers <= 0 && c.SystemMode == PrivateSystem {
		list = append(list, "please set attackers > 0")
	}
	if c.AttackerTimeout <= 0 {
		list = append(list, "please set attacker timeout > 0, seconds")
	}
	if c.StartRPS <= 0 {
		list = append(list, "please set start rps > 0")
	}
	if c.StepDurationSec < 0 {
		list = append(list, "please set step duration > 0, seconds")
	}
	if c.StepRPS < 0 {
		list = append(list, "please set step rps > 0")
	}
	if c.TestTimeSec <= 0 {
		list = append(list, "please set test time rps > 0, seconds")
	}
	return
}
