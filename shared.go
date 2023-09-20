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
