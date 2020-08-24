/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"sync"
)

type SharedDataSlice struct {
	*sync.Mutex
	Index int
	Data  []interface{}
}

func NewSharedDataSlice(data []interface{}) *SharedDataSlice {
	return &SharedDataSlice{
		Mutex: &sync.Mutex{},
		Index: 0,
		Data:  data,
	}
}

func (m *SharedDataSlice) Get() interface{} {
	m.Lock()
	if m.Index > len(m.Data)-1 {
		m.Index = 0
	}
	data := m.Data[m.Index]
	m.Index++
	m.Unlock()
	return data
}

func (m *SharedDataSlice) Add(d interface{}) {
	m.Lock()
	defer m.Unlock()
	m.Data = append(m.Data, d)
}
