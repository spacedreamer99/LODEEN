package net

import (
	"testing"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// --- payContractPrice ---

func TestPayContractPrice_Gather4_HasFruit(t *testing.T) {
	c := newTestClient("p1")
	c.inventory["fruit"] = 2

	if !payContractPrice(c, "gather4") {
		t.Fatalf("expected true with fruit")
	}
	if c.inventory["fruit"] != 1 {
		t.Fatalf("fruit = %d, want 1", c.inventory["fruit"])
	}
}

func TestPayContractPrice_Gather4_NoFruit(t *testing.T) {
	c := newTestClient("p1")

	if payContractPrice(c, "gather4") {
		t.Fatalf("expected false without fruit")
	}
}

func TestPayContractPrice_Guard_HasEnoughSpears(t *testing.T) {
	c := newTestClient("p1")
	c.inventory["spear"] = 15

	if !payContractPrice(c, "guard") {
		t.Fatalf("expected true with 15 spears")
	}
	if c.inventory["spear"] != 5 {
		t.Fatalf("spear = %d, want 5 (15-10)", c.inventory["spear"])
	}
}

func TestPayContractPrice_Guard_ExactlyTen(t *testing.T) {
	c := newTestClient("p1")
	c.inventory["spear"] = 10

	if !payContractPrice(c, "guard") {
		t.Fatalf("expected true with exactly 10 spears")
	}
	if c.inventory["spear"] != 0 {
		t.Fatalf("spear = %d, want 0", c.inventory["spear"])
	}
}

func TestPayContractPrice_Guard_NotEnough(t *testing.T) {
	c := newTestClient("p1")
	c.inventory["spear"] = 9

	if payContractPrice(c, "guard") {
		t.Fatalf("expected false with 9 spears")
	}
	if c.inventory["spear"] != 9 {
		t.Fatalf("spear should be unchanged")
	}
}

// --- applyContract ---

func TestApplyContract_Gather4_ResetsInventory(t *testing.T) {
	m := &Mob{ID: "m1", Inventory: map[string]int{"stone": 5}}

	applyContract(m, "p1", "gather4")

	if m.Contract != "gather4" {
		t.Fatalf("contract = %q", m.Contract)
	}
	if m.OwnerID != "p1" {
		t.Fatalf("owner = %q", m.OwnerID)
	}
	if len(m.Inventory) != 0 {
		t.Fatalf("inventory should be reset, got %+v", m.Inventory)
	}
}

func TestApplyContract_Guard_KeepsInventory(t *testing.T) {
	m := &Mob{ID: "m1", Inventory: map[string]int{"stone": 5}}

	applyContract(m, "p1", "guard")

	if m.Contract != "guard" {
		t.Fatalf("contract = %q", m.Contract)
	}
	if m.Inventory["stone"] != 5 {
		t.Fatalf("guard should not reset inventory")
	}
}

// --- tryAcceptContract ---

func TestTryAcceptContract_SuccessGather4(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["fruit"] = 1

	m := &Mob{ID: "m1", Kind: "pink", Inventory: map[string]int{}}
	s.mobs.Put("m1", m)

	if !s.tryAcceptContract(c, "m1", "gather4") {
		t.Fatalf("expected true")
	}
	if m.Contract != "gather4" || m.OwnerID != "p1" {
		t.Fatalf("contract not applied: %+v", m)
	}
	if c.inventory["fruit"] != 0 {
		t.Fatalf("fruit should be consumed")
	}
}

func TestTryAcceptContract_SuccessGuard(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["spear"] = 12

	m := &Mob{ID: "m1", Kind: "pink"}
	s.mobs.Put("m1", m)

	if !s.tryAcceptContract(c, "m1", "guard") {
		t.Fatalf("expected true")
	}
	if c.inventory["spear"] != 2 {
		t.Fatalf("spear = %d, want 2", c.inventory["spear"])
	}
}

func TestTryAcceptContract_MobNotFound(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")

	if s.tryAcceptContract(c, "nonexistent", "gather4") {
		t.Fatalf("expected false")
	}
}

func TestTryAcceptContract_NotPink(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["fruit"] = 1

	m := &Mob{ID: "m1", Kind: "hostile"}
	s.mobs.Put("m1", m)

	if s.tryAcceptContract(c, "m1", "gather4") {
		t.Fatalf("expected false for non-pink mob")
	}
	if c.inventory["fruit"] != 1 {
		t.Fatalf("fruit should not be consumed")
	}
}

func TestTryAcceptContract_AlreadyBusy(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["fruit"] = 1

	m := &Mob{ID: "m1", Kind: "pink", Contract: "guard"}
	s.mobs.Put("m1", m)

	if s.tryAcceptContract(c, "m1", "gather4") {
		t.Fatalf("expected false for busy mob")
	}
	if c.inventory["fruit"] != 1 {
		t.Fatalf("fruit should not be consumed")
	}
	if m.Contract != "guard" {
		t.Fatalf("existing contract should remain")
	}
}

func TestTryAcceptContract_UnknownContract(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["fruit"] = 1

	m := &Mob{ID: "m1", Kind: "pink"}
	s.mobs.Put("m1", m)

	if s.tryAcceptContract(c, "m1", "unknown-contract") {
		t.Fatalf("expected false for unknown contract")
	}
	if m.Contract != "" {
		t.Fatalf("contract should not be set")
	}
}

func TestTryAcceptContract_NoFruit(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")

	m := &Mob{ID: "m1", Kind: "pink"}
	s.mobs.Put("m1", m)

	if s.tryAcceptContract(c, "m1", "gather4") {
		t.Fatalf("expected false without fruit")
	}
	if m.Contract != "" {
		t.Fatalf("contract should not be set")
	}
}

func TestTryAcceptContract_NotEnoughSpears(t *testing.T) {
	s := newTestServer()
	c := newTestClient("p1")
	c.inventory["spear"] = 5

	m := &Mob{ID: "m1", Kind: "pink"}
	s.mobs.Put("m1", m)

	if s.tryAcceptContract(c, "m1", "guard") {
		t.Fatalf("expected false with 5 spears")
	}
	if c.inventory["spear"] != 5 {
		t.Fatalf("spear should be unchanged")
	}
}

// --- findNearestPlayer (чистая функция) ---

func TestFindNearestPlayer_Empty(t *testing.T) {
	got, _ := findNearestPlayer(nil, protocol.Vector3{})
	if got != nil {
		t.Fatalf("expected nil for empty players")
	}
}

func TestFindNearestPlayer_OutOfRange(t *testing.T) {
	players := []mobPlayerInfo{
		{state: protocol.PlayerState{X: 100, Y: 0, Z: 0}},
	}
	got, _ := findNearestPlayer(players, protocol.Vector3{})
	if got != nil {
		t.Fatalf("expected nil outside aggro radius")
	}
}

func TestFindNearestPlayer_Nearest(t *testing.T) {
	players := []mobPlayerInfo{
		{state: protocol.PlayerState{X: 20, Y: 0, Z: 0}},
		{state: protocol.PlayerState{X: 3, Y: 0, Z: 0}},
		{state: protocol.PlayerState{X: 10, Y: 0, Z: 0}},
	}
	got, d2 := findNearestPlayer(players, protocol.Vector3{})
	if got == nil {
		t.Fatalf("expected a player")
	}
	if got.state.X != 3 {
		t.Fatalf("expected X=3, got X=%v", got.state.X)
	}
	if d2 != 9 {
		t.Fatalf("d2 = %v, want 9", d2)
	}
}
