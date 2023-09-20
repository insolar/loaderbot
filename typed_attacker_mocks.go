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
