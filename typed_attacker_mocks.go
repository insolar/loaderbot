/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"context"
)

func init() {
	RegisterAttacker("TypedAttackerMock1", &TypedAttackerMock1{})
}

type TypedAttackerMock1 struct {
	*Runner
}

func (a *TypedAttackerMock1) Setup(c RunnerConfig) error {
	return nil
}

func (a *TypedAttackerMock1) Do(_ context.Context) DoResult {
	a.L.Infof("attack from type #1")
	return DoResult{RequestLabel: a.Name}
}

func (a *TypedAttackerMock1) Teardown() error {
	return nil
}

func (a *TypedAttackerMock1) Clone(r *Runner) Attack {
	return &TypedAttackerMock1{r}
}
