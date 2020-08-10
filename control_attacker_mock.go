/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"context"
	"log"
	"sync/atomic"
	"time"
)

type ControlAttackerMock struct {
	name         int
	serviceError chan bool
	r            *Runner
}

func (a *ControlAttackerMock) Clone(r *Runner) Attack {
	return &ControlAttackerMock{
		name:         1,
		serviceError: make(chan bool),
		r:            r,
	}
}

func (a *ControlAttackerMock) Setup(c RunnerConfig) error {
	return nil
}

func (a *ControlAttackerMock) Do(_ context.Context) DoResult {
	select {
	case <-a.serviceError:
		log.Printf("service error happens")
		return DoResult{RequestLabel: a.r.Name, Error: "service error"}
	default:
	}
	sleepTime := atomic.LoadInt64(&a.r.controlled.Sleep)
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	return DoResult{RequestLabel: a.r.Name}
}

func (a *ControlAttackerMock) Teardown() error {
	return nil
}

func NewControlMockAttacker(name int, serviceError chan bool, r *Runner) *ControlAttackerMock {
	return &ControlAttackerMock{name, serviceError, r}
}
