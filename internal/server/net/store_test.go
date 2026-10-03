package net

import (
	"sync"
	"testing"
)

// --- базовые операции ---

func TestStore_PutGet(t *testing.T) {
	s := newStore[int]()
	s.Put("a", 1)
	s.Put("b", 2)

	if v, ok := s.Get("a"); !ok || v != 1 {
		t.Fatalf("Get(a) = (%v,%v), want (1,true)", v, ok)
	}
	if v, ok := s.Get("b"); !ok || v != 2 {
		t.Fatalf("Get(b) = (%v,%v), want (2,true)", v, ok)
	}
	if _, ok := s.Get("missing"); ok {
		t.Fatalf("Get(missing) returned ok=true")
	}
}

func TestStore_PutOverwrites(t *testing.T) {
	s := newStore[string]()
	s.Put("k", "v1")
	s.Put("k", "v2")
	if v, _ := s.Get("k"); v != "v2" {
		t.Fatalf("overwrite failed: got %q, want %q", v, "v2")
	}
}

func TestStore_Delete(t *testing.T) {
	s := newStore[int]()
	s.Put("x", 42)
	s.Delete("x")
	if _, ok := s.Get("x"); ok {
		t.Fatalf("Delete did not remove key")
	}
	// Delete несуществующего ключа не паникует
	s.Delete("nonexistent")
}

func TestStore_Len(t *testing.T) {
	s := newStore[int]()
	if s.Len() != 0 {
		t.Fatalf("empty store Len = %d, want 0", s.Len())
	}
	s.Put("a", 1)
	s.Put("b", 2)
	s.Put("c", 3)
	if s.Len() != 3 {
		t.Fatalf("Len = %d, want 3", s.Len())
	}
	s.Delete("b")
	if s.Len() != 2 {
		t.Fatalf("after delete Len = %d, want 2", s.Len())
	}
}

// --- Read / Update ---

func TestStore_Read(t *testing.T) {
	s := newStore[int]()
	s.Put("a", 1)
	s.Put("b", 2)

	sum := 0
	s.Read(func(m map[string]int) {
		for _, v := range m {
			sum += v
		}
	})
	if sum != 3 {
		t.Fatalf("Read sum = %d, want 3", sum)
	}
}

func TestStore_Update(t *testing.T) {
	s := newStore[int]()
	s.Put("a", 1)
	s.Update(func(m map[string]int) {
		m["b"] = 2
		m["a"] = 100
	})
	if v, _ := s.Get("a"); v != 100 {
		t.Fatalf("Update did not write a: got %d", v)
	}
	if v, _ := s.Get("b"); v != 2 {
		t.Fatalf("Update did not write b: got %d", v)
	}
}

// --- Map: доступ под lock ---

func TestStore_MapUnderLock(t *testing.T) {
	s := newStore[int]()
	s.Put("a", 1)

	s.Lock()
	n := len(s.Map())
	s.Unlock()
	if n != 1 {
		t.Fatalf("Map() under lock: len = %d, want 1", n)
	}
}

// --- store указателей: изменение по ссылке ---

type fakeRocket struct{ Fuel int }

func TestStore_PointerSemantics(t *testing.T) {
	s := newStore[*fakeRocket]()
	s.Put("r1", &fakeRocket{Fuel: 100})

	got, _ := s.Get("r1")
	got.Fuel = 50

	again, _ := s.Get("r1")
	if again.Fuel != 50 {
		t.Fatalf("pointer not shared: got Fuel=%d, want 50", again.Fuel)
	}
}

// --- Конкурентный доступ: поймает race через -race ---

func TestStore_Concurrent(t *testing.T) {
	const goroutines = 8
	const iterations = 1000

	s := newStore[int]()

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Writers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				s.Put("shared", j)
			}
		}()
	}
	// Readers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				s.Get("shared")
			}
		}()
	}
	// Read/Update через функциональный API
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				s.Read(func(m map[string]int) {
					_ = len(m)
				})
				s.Update(func(m map[string]int) {
					m["counter"] = j
				})
			}
		}()
	}

	wg.Wait()
}
