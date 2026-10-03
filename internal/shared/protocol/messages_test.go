package protocol

import (
	"encoding/json"
	"math"
	"testing"
)

// --- Type constants ---

// TestTypeConstants_UniqueAndNonEmpty ловит опечатки при добавлении новых типов.
func TestTypeConstants_UniqueAndNonEmpty(t *testing.T) {
	all := []Type{
		TypeHello, TypeWelcome, TypeState, TypeSnapshot, TypeChat,
		TypePing, TypePong, TypePickupItem, TypeInventoryUpdate, TypeEatFruit,
		TypeThrowSpear, TypeCraftItem, TypeHitMammoth, TypePlantSeed, TypeWaterPlant,
		TypeTakeWater, TypeTameMammoth, TypeSelectItem, TypeLeashMammoth, TypePlaceHouse,
		TypePlaceSolar, TypePlaceBattery, TypePlaceFactory, TypeToggleDoor, TypeSaddleMammoth,
		TypeRideMammoth, TypePlaceBoat, TypeEnterBoat, TypeHitMob, TypeAcceptContract,
		TypeOpenFactory, TypeCraftFactory, TypePlaceRocket, TypeBoardRocket, TypeExitRocket,
		TypeRocketInput, TypeTeleport,
	}
	seen := make(map[Type]struct{}, len(all))
	for _, tp := range all {
		if tp == "" {
			t.Fatalf("empty Type constant in list")
		}
		if _, dup := seen[tp]; dup {
			t.Fatalf("duplicate Type: %q", tp)
		}
		seen[tp] = struct{}{}
	}
}

// --- Envelope ---

func TestEnvelopeRoundTrip_PlayerState(t *testing.T) {
	in := PlayerState{
		ID: "p1", Nick: "alice",
		X: 1.5, Y: -2.5, Z: 3.5,
		Yaw: 0.1, Pitch: -0.2,
		Hunger: 42.5, HP: 100, RTTms: 55,
	}
	env, err := NewEnvelope(TypeState, in)
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	if env.Type != TypeState {
		t.Fatalf("type = %q, want %q", env.Type, TypeState)
	}
	var out PlayerState
	if err := env.Decode(&out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestEnvelope_DecodeEmptyPayload(t *testing.T) {
	env := &Envelope{Type: TypeExitRocket}
	var out ExitRocket
	if err := env.Decode(&out); err != nil {
		t.Fatalf("Decode on empty payload: %v", err)
	}
}

func TestEnvelope_DecodeInvalidJSON(t *testing.T) {
	env := &Envelope{Type: TypeChat, Data: json.RawMessage(`{not-json`)}
	var out ChatMessage
	if err := env.Decode(&out); err == nil {
		t.Fatalf("expected error decoding invalid JSON, got nil")
	}
}

func TestNewEnvelope_MarshalError(t *testing.T) {
	if _, err := NewEnvelope(TypeState, math.NaN()); err == nil {
		t.Fatalf("expected marshal error for NaN, got nil")
	}
}

// --- Vector3 ---

func TestVector3_MarshalUnmarshal(t *testing.T) {
	in := Vector3{X: 1, Y: 2, Z: 3}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Vector3
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch: %+v != %+v", in, out)
	}
}

// --- Map payload ---

func TestInventoryUpdate_MapRoundTrip(t *testing.T) {
	in := InventoryUpdate{Items: map[string]int{
		"stone": 10, "wood": 5, "rocket": 1,
	}}
	env, err := NewEnvelope(TypeInventoryUpdate, in)
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	var out InventoryUpdate
	if err := env.Decode(&out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out.Items) != len(in.Items) {
		t.Fatalf("len mismatch: %d vs %d", len(out.Items), len(in.Items))
	}
	for k, v := range in.Items {
		if out.Items[k] != v {
			t.Fatalf("item %q = %d, want %d", k, out.Items[k], v)
		}
	}
}

// --- Chat ---

func TestChatMessage_RoundTrip(t *testing.T) {
	in := ChatMessage{From: "alice", Text: "hello, world", TS: 1700000000}
	env, err := NewEnvelope(TypeChat, in)
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	var out ChatMessage
	if err := env.Decode(&out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch: %+v != %+v", in, out)
	}
}

// --- Rocket ---

func TestRocket_FullRoundTrip(t *testing.T) {
	in := Rocket{
		ID: "r1", X: 1, Y: 2, Z: 3, DX: 0, DY: 1, DZ: 0,
		Fuel: 100, MaxFuel: 100, Piloted: true, OwnerID: "p1",
		InOrbit: true, Apoapsis: 500, Periapsis: 400,
		Speed: 7.5, Altitude: 350, TargetVelocity: 7.8,
		VX: 1, VY: 2, VZ: 3, PrimaryBody: "earth",
		DistSun: 1e8, SOIRadius: 5e5,
	}
	env, err := NewEnvelope(TypeSnapshot, in)
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	var out Rocket
	if err := env.Decode(&out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

// --- Snapshot (полный, все домены) ---

func TestSnapshot_FullRoundTrip(t *testing.T) {
	in := Snapshot{
		Tick:      12345,
		Players:   []PlayerState{{ID: "p1", Nick: "a", X: 1, Y: 2, Z: 3, HP: 100}},
		Resources: []Resource{{ID: "r1", Type: "stone", X: 1, Y: 2, Z: 3}},
		Mammoths:  []Mammoth{{ID: "m1", X: 1, Y: 2, Z: 3, HP: 5, Sex: "f"}},
		Wells:     []Well{{ID: "w1", X: 1, Y: 2, Z: 3}},
		Houses:    []House{{ID: "h1", X: 1, Y: 2, Z: 3, DoorOpen: true}},
		Boats:     []Boat{{ID: "b1", X: 1, Y: 2, Z: 3, RiderID: "p1"}},
		Mobs:      []Mob{{ID: "mob1", X: 1, Y: 2, Z: 3, HP: 5, Kind: "hostile"}},
		Projectiles: []MobProjectile{
			{ID: "pr1", X: 1, Y: 2, Z: 3, DX: 1, DY: 0, DZ: 0},
		},
		Solar:      []Solar{{ID: "s1", X: 1, Y: 2, Z: 3}},
		Batteries:  []Battery{{ID: "bat1", X: 1, Y: 2, Z: 3, Energy: 100, MaxEnergy: 1000}},
		Factories:  []Factory{{ID: "f1", X: 1, Y: 2, Z: 3, Crafting: "steel", Progress: 50}},
		Rockets:    []Rocket{{ID: "rk1", X: 1, Y: 2, Z: 3, Fuel: 50, MaxFuel: 100, Piloted: true}},
		EarthPos:   Vector3{X: 10, Y: 20, Z: 30},
		EarthVel:   Vector3{X: 0.1, Y: 0.2, Z: 0.3},
		Planet2Pos: Vector3{X: 100, Y: 200, Z: 300},
		Planet2Vel: Vector3{X: 1, Y: 2, Z: 3},
	}
	env, err := NewEnvelope(TypeSnapshot, in)
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	var out Snapshot
	if err := env.Decode(&out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out.Tick != in.Tick {
		t.Fatalf("tick: %d vs %d", out.Tick, in.Tick)
	}
	if len(out.Players) != 1 || out.Players[0].ID != "p1" {
		t.Fatalf("players mismatch: %+v", out.Players)
	}
	if out.EarthPos != in.EarthPos {
		t.Fatalf("earth pos mismatch: %+v vs %+v", out.EarthPos, in.EarthPos)
	}
	if len(out.Rockets) != 1 || !out.Rockets[0].Piloted {
		t.Fatalf("rockets mismatch: %+v", out.Rockets)
	}
	if len(out.Factories) != 1 || out.Factories[0].Crafting != "steel" {
		t.Fatalf("factories mismatch: %+v", out.Factories)
	}
}
