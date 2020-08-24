/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package attackers

import (
	"context"
	"net/http"

	"github.com/insolar/loaderbot"
)

type DataAttackerExample struct {
	*loaderbot.Runner
	client *http.Client
}

func (a *DataAttackerExample) Clone(r *loaderbot.Runner) loaderbot.Attack {
	return &DataAttackerExample{Runner: r}
}

func (a *DataAttackerExample) Setup(c loaderbot.RunnerConfig) error {
	a.client = loaderbot.NewLoggingHTTPClient(c.DumpTransport, 10)
	return nil
}

func (a *DataAttackerExample) Do(_ context.Context) loaderbot.DoResult {
	data := a.TestData.(*loaderbot.SharedDataSlice).Get()
	a.Runner.L.Infof("firing with data: %s", data)
	_, err := a.client.Get(a.Cfg.TargetUrl)
	if err != nil {
		return loaderbot.DoResult{
			RequestLabel: a.Name,
			Error:        err.Error(),
		}
	}
	return loaderbot.DoResult{
		RequestLabel: a.Name,
	}
}

func (a *DataAttackerExample) Teardown() error {
	return nil
}
