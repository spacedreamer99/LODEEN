package net

import (
	"io"
	"log/slog"
	"testing"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// newTestServer собирает минимальный Server со всеми доменными store'ами,
// но без listener'а, метрик и spawn'а сущностей. Достаточно для юнит-тестов
// на handlers, tick-фазы и валидаторы.
func newTestServer() *Server {
	return &Server{
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		clients:     make(map[string]*Client),
		resources:   newStore[protocol.Resource](),
		mammoths:    newStore[*Mammoth](),
		wells:       newStore[*protocol.Well](),
		houses:      newStore[*House](),
		solar:       newStore[*Solar](),
		batteries:   newStore[*Battery](),
		factories:   newStore[*Factory](),
		rockets:     newStore[*Rocket](),
		boats:       newStore[*Boat](),
		mobs:        newStore[*Mob](),
		projectiles: newStore[*Projectile](),
		done:        make(chan struct{}),
		world:       NewWorldState(),
	}
}

// newTestClient создаёт клиента без TCP-соединения.
// send-канал буферизован и опустошается горутиной-дренажом,
// чтобы sendEnvelope не блокировался в тестах.
func newTestClient(id string) *Client {
	c := &Client{
		ID:        id,
		Nick:      "test-" + id,
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		inventory: make(map[string]int),
		hunger:    100,
		hp:        100,
	}
	go func() {
		for {
			select {
			case <-c.send:
			case <-c.done:
				return
			}
		}
	}()
	return c
}

// firstRecipe возвращает произвольный рецепт из factoryRecipes.
// Используется в тестах, где важен сам факт «какой-то рецепт», а не конкретный.
func firstRecipe(t *testing.T) (string, factoryRecipe) {
	t.Helper()
	for k, v := range factoryRecipes {
		return k, v
	}
	t.Fatal("no factory recipes registered")
	return "", factoryRecipe{}
}
