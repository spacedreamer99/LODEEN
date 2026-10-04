package net

import (
	"testing"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// --- applyHitMob ---

func TestApplyHitMob_Success(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["spear"] = 3

	// Моб прямо на игроке (в пределах 100м).
	c.setState(protocol.PlayerState{ID: "p1", X: 0, Y: 0, Z: 0})
	m := &Mob{ID: "m1", Pos: protocol.Vector3{X: 5, Y: 0, Z: 0}, HP: 3, Kind: "hostile"}
	s.mobs.Put("m1", m)

	res, ok := s.applyHitMob(c, "m1", c.State())
	if !ok {
		t.Fatalf("applyHitMob returned false")
	}
	if res.killed {
		t.Fatalf("mob should not be killed (HP 3 -> 2)")
	}
	if res.hpAfter != 2 {
		t.Fatalf("hpAfter = %d, want 2", res.hpAfter)
	}
	if c.inventory["spear"] != 2 {
		t.Fatalf("spear = %d, want 2 (one consumed)", c.inventory["spear"])
	}
	if _, exists := s.mobs.Map()["m1"]; !exists {
		t.Fatalf("mob should still exist")
	}
}

func TestApplyHitMob_KillsMob(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["spear"] = 1

	c.setState(protocol.PlayerState{ID: "p1", X: 0, Y: 0, Z: 0})
	m := &Mob{ID: "m1", Pos: protocol.Vector3{X: 5, Y: 0, Z: 0}, HP: 1, Kind: "hostile"}
	s.mobs.Put("m1", m)

	res, ok := s.applyHitMob(c, "m1", c.State())
	if !ok {
		t.Fatalf("applyHitMob returned false")
	}
	if !res.killed {
		t.Fatalf("mob should be killed")
	}
	if res.kind != "hostile" {
		t.Fatalf("kind = %q, want hostile", res.kind)
	}
	if _, exists := s.mobs.Map()["m1"]; exists {
		t.Fatalf("killed mob should be removed")
	}
	if res.drop.pos != (protocol.Vector3{X: 5, Y: 0, Z: 0}) {
		t.Fatalf("drop.pos = %+v, want {5,0,0}", res.drop.pos)
	}
}

func TestApplyHitMob_TooFar(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["spear"] = 1

	c.setState(protocol.PlayerState{ID: "p1", X: 0, Y: 0, Z: 0})
	// > 100м
	m := &Mob{ID: "m1", Pos: protocol.Vector3{X: 200, Y: 0, Z: 0}, HP: 3, Kind: "hostile"}
	s.mobs.Put("m1", m)

	_, ok := s.applyHitMob(c, "m1", c.State())
	if ok {
		t.Fatalf("expected false for out-of-range mob")
	}
	if c.inventory["spear"] != 1 {
		t.Fatalf("spear should not be consumed")
	}
	if m.HP != 3 {
		t.Fatalf("mob HP should be unchanged")
	}
}

func TestApplyHitMob_NotFound(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["spear"] = 1

	_, ok := s.applyHitMob(c, "nonexistent", c.State())
	if ok {
		t.Fatalf("expected false for unknown mob")
	}
	if c.inventory["spear"] != 1 {
		t.Fatalf("spear should not be consumed")
	}
}

func TestApplyHitMob_NoSpear(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	// пустой инвентарь

	c.setState(protocol.PlayerState{ID: "p1"})
	m := &Mob{ID: "m1", Pos: protocol.Vector3{X: 5}, HP: 3, Kind: "hostile"}
	s.mobs.Put("m1", m)

	_, ok := s.applyHitMob(c, "m1", c.State())
	if ok {
		t.Fatalf("expected false without spear")
	}
	if m.HP != 3 {
		t.Fatalf("mob HP should be unchanged")
	}
}

func TestApplyHitMob_CollectorAngered(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["spear"] = 1
	c.setState(protocol.PlayerState{ID: "p1"})

	m := &Mob{ID: "m1", Pos: protocol.Vector3{X: 5}, HP: 10, Kind: "collector"}
	s.mobs.Put("m1", m)

	_, ok := s.applyHitMob(c, "m1", c.State())
	if !ok {
		t.Fatalf("applyHitMob returned false")
	}
	if !m.Angered {
		t.Fatalf("collector should become Angered after hit")
	}
}

// --- applyHitMammoth ---

func TestApplyHitMammoth_Success(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.setState(protocol.PlayerState{ID: "p1", X: 0, Y: 0, Z: 0})

	m := &Mammoth{ID: "m1", Pos: protocol.Vector3{X: 5, Y: 0, Z: 0}, HP: 3}
	s.mammoths.Put("m1", m)

	res, ok := s.applyHitMammoth(c, "m1")
	if !ok {
		t.Fatalf("applyHitMammoth returned false")
	}
	if res.killed {
		t.Fatalf("mammoth should not be killed")
	}
	if res.hpAfter != 2 {
		t.Fatalf("hpAfter = %d, want 2", res.hpAfter)
	}
}

func TestApplyHitMammoth_Killed(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.setState(protocol.PlayerState{ID: "p1", X: 0, Y: 0, Z: 0})

	m := &Mammoth{ID: "m1", Pos: protocol.Vector3{X: 5, Y: 0, Z: 0}, HP: 1}
	s.mammoths.Put("m1", m)

	res, ok := s.applyHitMammoth(c, "m1")
	if !ok {
		t.Fatalf("applyHitMammoth returned false")
	}
	if !res.killed {
		t.Fatalf("mammoth should be killed")
	}
	if _, exists := s.mammoths.Map()["m1"]; exists {
		t.Fatalf("killed mammoth should be removed")
	}
	if res.pos != (protocol.Vector3{X: 5, Y: 0, Z: 0}) {
		t.Fatalf("drop pos = %+v", res.pos)
	}
}

func TestApplyHitMammoth_TooFar(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.setState(protocol.PlayerState{ID: "p1"})

	// > 80м
	m := &Mammoth{ID: "m1", Pos: protocol.Vector3{X: 200}, HP: 3}
	s.mammoths.Put("m1", m)

	_, ok := s.applyHitMammoth(c, "m1")
	if ok {
		t.Fatalf("expected false for out-of-range mammoth")
	}
	if m.HP != 3 {
		t.Fatalf("mammoth HP should be unchanged")
	}
}

func TestApplyHitMammoth_NotFound(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")

	_, ok := s.applyHitMammoth(c, "nonexistent")
	if ok {
		t.Fatalf("expected false for unknown mammoth")
	}
}

// --- handleHitMammoth: debounce + throw window ---

func TestHandleHitMammoth_Debounce(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["spear"] = 5
	c.setState(protocol.PlayerState{ID: "p1"})

	// Уже бил 50мс назад → следующий удар игнорируется.
	c.lastHitAt = time.Now()
	c.lastThrowAt = time.Now() // throw был недавно

	m := &Mammoth{ID: "m1", Pos: protocol.Vector3{X: 5}, HP: 3}
	s.mammoths.Put("m1", m)

	s.handleHitMammoth(c, "m1")
	if m.HP != 3 {
		t.Fatalf("debounced hit should not damage: HP = %d", m.HP)
	}
}

func TestHandleHitMammoth_NoRecentThrow(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.setState(protocol.PlayerState{ID: "p1"})

	// Последний бросок был 5 секунд назад → удар отклоняется.
	c.lastHitAt = time.Time{}
	c.lastThrowAt = time.Now().Add(-5 * time.Second)

	m := &Mammoth{ID: "m1", Pos: protocol.Vector3{X: 5}, HP: 3}
	s.mammoths.Put("m1", m)

	s.handleHitMammoth(c, "m1")
	if m.HP != 3 {
		t.Fatalf("hit without recent throw should not damage: HP = %d", m.HP)
	}
}
