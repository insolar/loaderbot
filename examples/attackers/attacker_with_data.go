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
	"sync"

	"github.com/insolar/loaderbot"
)

type SharedData struct {
	*sync.Mutex
	Index int
	Data  []string
}

func (m *SharedData) GetNextData() string {
	m.Lock()
	if m.Index > len(m.Data)-1 {
		m.Index = 0
	}
	data := m.Data[m.Index]
	m.Index++
	m.Unlock()
	return data
}

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
	data := a.TestData.(*SharedData).GetNextData()
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
