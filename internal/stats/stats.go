package stats

import (
	"fmt"
	"sync"
)

type Stats struct {
	mu     sync.Mutex
	data   map[string]int
	active bool
}

func New(active bool) *Stats {
	return &Stats{
		data:   make(map[string]int),
		active: active,
	}
}

func (s *Stats) Update(key string) {
	if !s.active {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key]++
}

func (s *Stats) Print(f func(format string, args ...interface{}), elapsed string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}
	if len(s.data) == 0 {
		f("No corrective actions applied in the last %s", elapsed)
		return
	}
	msg := fmt.Sprintf("What happened in the last %s:", elapsed)
	for k, v := range s.data {
		msg += fmt.Sprintf("\n  %s: %d", k, v)
	}
	f(msg)
}
