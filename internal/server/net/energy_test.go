package net

import (
	"testing"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// --- tryCraftFactory: атомарный крафт ---

func TestTryCraftFactory_Success(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")

	_, r := firstRecipe(t)

	// Даём ресурсов «с запасом», чтобы проверить точное списание.
	const spare = 5
	for item, q := range r.need {
		c.inventory[item] = q + spare
	}

	f := &Factory{ID: "f1", Pos: protocol.Vector3{X: 0, Y: 0, Z: 0}}
	b := &Battery{ID: "b1", Pos: protocol.Vector3{X: 1, Y: 0, Z: 0}, Energy: 1000, MaxEnergy: 1000}
	s.factories.Put("f1", f)
	s.batteries.Put("b1", b)

	ok := s.tryCraftFactory(c, f, r)
	if !ok {
		t.Fatalf("tryCraftFactory returned false; want true")
	}

	if b.Energy != 1000-r.energy {
		t.Fatalf("battery energy = %d, want %d", b.Energy, 1000-r.energy)
	}
	for item := range r.need {
		want := spare
		if item == r.out {
			want = spare + r.outQty
		}
		if c.inventory[item] != want {
			t.Fatalf("inventory[%q] = %d, want %d", item, c.inventory[item], want)
		}
	}
}

func TestTryCraftFactory_NoResources(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	_, r := firstRecipe(t)

	// Пустой инвентарь.
	f := &Factory{ID: "f1"}
	b := &Battery{ID: "b1", Pos: protocol.Vector3{}, Energy: 1000, MaxEnergy: 1000}
	s.factories.Put("f1", f)
	s.batteries.Put("b1", b)

	ok := s.tryCraftFactory(c, f, r)
	if ok {
		t.Fatalf("expected false with empty inventory")
	}

	// Энергия не должна быть списана.
	if b.Energy != 1000 {
		t.Fatalf("battery energy = %d, want 1000 (unchanged)", b.Energy)
	}
}

func TestTryCraftFactory_PartialResources(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	_, r := firstRecipe(t)

	// Даём на 1 меньше каждого ресурса.
	for item, q := range r.need {
		c.inventory[item] = q - 1
	}

	f := &Factory{ID: "f1"}
	b := &Battery{ID: "b1", Pos: protocol.Vector3{}, Energy: 1000, MaxEnergy: 1000}
	s.factories.Put("f1", f)
	s.batteries.Put("b1", b)

	ok := s.tryCraftFactory(c, f, r)
	if ok {
		t.Fatalf("expected false with partial inventory")
	}
	if b.Energy != 1000 {
		t.Fatalf("battery energy = %d, want 1000 (unchanged)", b.Energy)
	}
}

func TestTryCraftFactory_NoBattery(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	_, r := firstRecipe(t)

	for item, q := range r.need {
		c.inventory[item] = q
	}
	f := &Factory{ID: "f1"}
	s.factories.Put("f1", f)
	// Батареи нет.

	ok := s.tryCraftFactory(c, f, r)
	if ok {
		t.Fatalf("expected false with no battery")
	}

	// Инвентарь не должен измениться.
	for item, q := range r.need {
		if c.inventory[item] != q {
			t.Fatalf("inventory[%q] changed despite failure: %d, want %d",
				item, c.inventory[item], q)
		}
	}
}

func TestTryCraftFactory_BatteryOutOfRange(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	_, r := firstRecipe(t)

	for item, q := range r.need {
		c.inventory[item] = q
	}

	f := &Factory{ID: "f1", Pos: protocol.Vector3{X: 0, Y: 0, Z: 0}}
	// energyLinkRadius = 10, батарея на расстоянии 100 — вне радиуса.
	b := &Battery{ID: "b1", Pos: protocol.Vector3{X: 100, Y: 0, Z: 0}, Energy: 1000, MaxEnergy: 1000}
	s.factories.Put("f1", f)
	s.batteries.Put("b1", b)

	ok := s.tryCraftFactory(c, f, r)
	if ok {
		t.Fatalf("expected false with out-of-range battery")
	}
	if b.Energy != 1000 {
		t.Fatalf("battery energy should be unchanged: %d", b.Energy)
	}
}

func TestTryCraftFactory_NotEnoughEnergy(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	_, r := firstRecipe(t)

	for item, q := range r.need {
		c.inventory[item] = q
	}

	f := &Factory{ID: "f1"}
	// Энергии меньше, чем нужно. r.energy >= 0 всегда, ставим 0 если рецепт без энергии.
	lowEnergy := r.energy - 1
	if lowEnergy < 0 {
		t.Skip("recipe has no energy cost; skip")
	}
	b := &Battery{ID: "b1", Pos: protocol.Vector3{}, Energy: lowEnergy, MaxEnergy: 1000}
	s.factories.Put("f1", f)
	s.batteries.Put("b1", b)

	ok := s.tryCraftFactory(c, f, r)
	if ok {
		t.Fatalf("expected false with insufficient energy")
	}
	if b.Energy != lowEnergy {
		t.Fatalf("battery energy should be unchanged: %d, want %d", b.Energy, lowEnergy)
	}
}

// --- lookupFactoryAndRecipe ---

func TestLookupFactoryAndRecipe_FactoryNotFound(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")

	_, _, ok := s.lookupFactoryAndRecipe(c, "nonexistent", "any")
	if ok {
		t.Fatalf("expected false with unknown factory")
	}
}

func TestLookupFactoryAndRecipe_RecipeNotFound(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")

	f := &Factory{ID: "f1"}
	s.factories.Put("f1", f)

	_, _, ok := s.lookupFactoryAndRecipe(c, "f1", "nonexistent-recipe")
	if ok {
		t.Fatalf("expected false with unknown recipe")
	}
}

func TestLookupFactoryAndRecipe_OK(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	recipeID, _ := firstRecipe(t)

	f := &Factory{ID: "f1"}
	s.factories.Put("f1", f)

	gotF, gotR, ok := s.lookupFactoryAndRecipe(c, "f1", recipeID)
	if !ok {
		t.Fatalf("expected ok")
	}
	if gotF != f {
		t.Fatalf("returned wrong factory")
	}
	if gotR.energy == 0 && gotR.out == "" {
		// Просто отметим, что рецепт получен.
	}
}
