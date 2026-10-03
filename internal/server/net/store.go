package net

import "sync"

// store — типизированный map с собственным RWMutex.
// Один store на домен (rockets, mobs, mammoths, ...).
//
// Использование:
//
//	s.rockets.Lock()
//	for _, r := range s.rockets.Map() { ... }
//	s.rockets.Unlock()
//
// Для типовых случаев см. Read/Update ниже — они делают defer сами.
type store[T any] struct {
	mu sync.RWMutex
	m  map[string]T
}

func newStore[T any]() *store[T] {
	return &store[T]{m: make(map[string]T)}
}

func (s *store[T]) Lock()    { s.mu.Lock() }
func (s *store[T]) Unlock()  { s.mu.Unlock() }
func (s *store[T]) RLock()   { s.mu.RLock() }
func (s *store[T]) RUnlock() { s.mu.RUnlock() }

// Map возвращает map для прямой итерации/доступа.
// ВАЖНО: вызывающий обязан держать Lock или RLock.
func (s *store[T]) Map() map[string]T { return s.m }

// Get — безопасное чтение с внутренним RLock.
func (s *store[T]) Get(id string) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[id]
	return v, ok
}

// Put — безопасная запись с внутренним Lock.
func (s *store[T]) Put(id string, v T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = v
}

// Delete — безопасное удаление.
func (s *store[T]) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
}

// Len — количество элементов.
func (s *store[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

// Read — короткий RLock на время fn.
func (s *store[T]) Read(fn func(m map[string]T)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s.m)
}

// Update — Lock на время fn.
func (s *store[T]) Update(fn func(m map[string]T)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.m)
}
