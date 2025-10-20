package concurrent

import "sync"

type Map[K comparable, V any] struct {
	mu     sync.RWMutex
	values map[K]V
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		values: make(map[K]V),
	}
}

func (m *Map[K, V]) Load(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.values[key]
	return val, ok
}

func (m *Map[K, V]) Store(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.values[key] = value
}

func (m *Map[K, V]) Length() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.values)
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for k, v := range m.values {
		if !f(k, v) {
			break
		}
	}
}
