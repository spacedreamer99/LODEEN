// Package net implements the LODEEN TCP game server.
package net

import (
	"crypto/rand"
	"time"
	"math"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

const (
	readTimeout  = 60 * time.Second
	writeTimeout = 5 * time.Second
	sendBufSize  = 256
	pickupRadius = 5.0
)

type Metrics struct {
	PlayersConnected prometheus.Gauge
	TicksTotal       prometheus.Counter
	ChatMessages     prometheus.Counter
}


type Server struct {
	addr     string
	tickRate int
	log      *slog.Logger
	metrics  *Metrics

	ln net.Listener

	mu      sync.RWMutex
	clients map[string]*Client

	resourcesMu sync.RWMutex
	resources   map[string]protocol.Resource

	mammothsMu sync.RWMutex
	mammoths   map[string]*Mammoth

	wellsMu sync.RWMutex
	wells   map[string]*protocol.Well

	housesMu sync.RWMutex
	houses   map[string]*House

	solarMu sync.RWMutex
	solar   map[string]*Solar

	batteriesMu sync.RWMutex
	batteries   map[string]*Battery

	factoriesMu sync.RWMutex
	factories   map[string]*Factory

	boatsMu sync.RWMutex
	boats   map[string]*Boat

	mobsMu sync.RWMutex
	mobs   map[string]*Mob

	projMu      sync.RWMutex
	projectiles map[string]*Projectile

	tick uint64

	wg        sync.WaitGroup
	closeOnce sync.Once
	done      chan struct{}
}

func New(addr string, tickRate int, log *slog.Logger, m *Metrics) *Server {
	s := &Server{
		addr:      addr,
		tickRate:  tickRate,
		log:       log,
		metrics:   m,
		clients:   make(map[string]*Client),
		resources: make(map[string]protocol.Resource),
		done:      make(chan struct{}),
	}
	s.mammoths = make(map[string]*Mammoth)
	s.spawnResources(40)
	s.spawnMammoths(5)

	s.wells = make(map[string]*protocol.Well)
	s.spawnWells(6)

	s.houses = make(map[string]*House)
	s.solar = make(map[string]*Solar)
	s.batteries = make(map[string]*Battery)
	s.factories = make(map[string]*Factory)
	s.boats = make(map[string]*Boat)
	s.mobs = make(map[string]*Mob)
	s.spawnMobs(5)
	s.projectiles = make(map[string]*Projectile)
	return s
}

func (s *Server) handleCraft(c *Client, recipe string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Требования рецептов — на сервере, для валидации.
	type req map[string]int
	recipes := map[string]struct {
		need req
		have req
	}{
		"spear":  {need: req{"stone": 2, "wood": 1}, have: req{"spear": 1}},
		"leash":  {need: req{"liana": 2}, have: req{"leash": 1}},
		"house":  {need: req{"wood": 50}, have: req{"house": 1}},
		"saddle": {need: req{"liana": 4}, have: req{"saddle": 1}},
		"boat":   {need: req{"wood": 20}, have: req{"boat": 1}},
	}
	r, ok := recipes[recipe]
	if !ok {
		c.log.Warn("unknown recipe", "recipe", recipe)
		return
	}
	for k, n := range r.need {
		if c.inventory[k] < n {
			c.log.Info("craft failed: not enough materials", "recipe", recipe, "missing", k)
			return
		}
	}
	for k, n := range r.need {
		c.inventory[k] -= n
		if c.inventory[k] == 0 {
			delete(c.inventory, k)
		}
	}
	for k, n := range r.have {
		c.inventory[k] += n
	}

	out := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		out[k] = v
	}
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: out})
	c.log.Info("crafted", "recipe", recipe)
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.ln = ln
	s.log.Info("tcp server listening", "addr", ln.Addr().String())
	s.wg.Add(2)
	go s.acceptLoop()
	go s.tickLoop()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.closeOnce.Do(func() {
		close(s.done)
		if s.ln != nil {
			_ = s.ln.Close()
		}
		s.mu.Lock()
		for _, c := range s.clients {
			_ = c.conn.Close()
		}
		s.mu.Unlock()
	})
	waitCh := make(chan struct{})
	go func() { s.wg.Wait(); close(waitCh) }()
	select {
	case <-waitCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) Addr() string {
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				s.log.Warn("accept error", "err", err)
				continue
			}
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()

	id := newID()
	log := s.log.With("id", id, "remote", conn.RemoteAddr().String())

	client := &Client{
		ID:        id,
		conn:      conn,
		send:      make(chan []byte, sendBufSize),
		done:      make(chan struct{}),
		log:       log,
		inventory: make(map[string]int),
		hunger:    100,
		hp:        100,
	}

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	env, err := readEnvelope(conn)
	if err != nil {
		log.Warn("hello read failed", "err", err)
		_ = conn.Close()
		return
	}
	if env.Type != protocol.TypeHello {
		log.Warn("expected hello", "got", string(env.Type))
		_ = conn.Close()
		return
	}
	var hello protocol.Hello
	if err := env.Decode(&hello); err != nil {
		log.Warn("hello decode failed", "err", err)
		_ = conn.Close()
		return
	}
	if hello.Nick == "" {
		hello.Nick = "anon-" + id[:6]
	}
	client.Nick = hello.Nick
	client.setState(protocol.PlayerState{ID: id, Nick: hello.Nick})
	_ = conn.SetReadDeadline(time.Time{})

	client.sendEnvelope(protocol.TypeWelcome, protocol.Welcome{
		PlayerID:      id,
		WorldName:     "lodeen-flat",
		TickRate:      s.tickRate,
		ServerVersion: "dev",
	})
	client.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{
		Items: map[string]int{},
	})

	s.mu.Lock()
	s.clients[id] = client
	s.mu.Unlock()
	s.metrics.PlayersConnected.Inc()
	log.Info("player joined", "nick", client.Nick)

	writerDone := make(chan struct{})
	go func() { defer close(writerDone); s.writeLoop(client) }()

	s.readLoop(client)

	client.closeOnce.Do(func() { close(client.done) })
	s.mu.Lock()
	delete(s.clients, id)
	s.mu.Unlock()
	s.metrics.PlayersConnected.Dec()
	_ = conn.Close()
	<-writerDone
	log.Info("player left", "nick", client.Nick)
}

func (s *Server) readLoop(c *Client) {
	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(readTimeout))
		env, err := readEnvelope(c.conn)
		if err != nil {
			return
		}
		s.handleMessage(c, env)
	}
}

func (s *Server) writeLoop(c *Client) {
	for {
		select {
		case payload := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := protocol.WriteFrame(c.conn, payload); err != nil {
				return
			}
		case <-c.done:
			return
		case <-s.done:
			return
		}
	}
}

func (s *Server) handleMessage(c *Client, env *protocol.Envelope) {
	switch env.Type {
	case protocol.TypeState:
		var st protocol.PlayerState
		if err := env.Decode(&st); err != nil {
			return
		}
		st.ID = c.ID
		st.Nick = c.Nick
		pos := protocol.ClampToSurface(protocol.Vector3{X: st.X, Y: st.Y, Z: st.Z})
		l := float32(math.Sqrt(float64(pos.X*pos.X + pos.Y*pos.Y + pos.Z*pos.Z)))
		if l > 0.01 {
			h := (l + protocol.PlayerHeight) / l
			pos.X *= h
			pos.Y *= h
			pos.Z *= h
		}
		st.X = pos.X
		st.Y = pos.Y
		st.Z = pos.Z
		c.setState(st)
	case protocol.TypeChat:
		var cm protocol.ChatMessage
		if err := env.Decode(&cm); err != nil {
			return
		}
		// Чит-команда /allinv — выдать все предметы по 100.
		if cm.Text == "/allinv" {
			allItems := []string{
				"stone", "wood", "ore", "fruit", "meat", "spear", "torch",
				"water", "liana", "leash", "house", "saddle", "boat",
				"solar", "battery", "factory",
				"steel", "gear", "circuit", "drone",
			}
			c.mu.Lock()
			for _, it := range allItems {
				c.inventory[it] = 100
			}
			inv := make(map[string]int, len(c.inventory))
			for k, v := range c.inventory {
				inv[k] = v
			}
			c.mu.Unlock()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("cheat: allinv", "count", len(allItems))
			break
		}
		// Чит-команда /clearinv — очистить инвентарь.
		if cm.Text == "/clearinv" {
			c.mu.Lock()
			c.inventory = make(map[string]int)
			inv := make(map[string]int)
			c.mu.Unlock()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("cheat: clearinv")
			break
		}
		// Чит-команды вида /get<item><qty>, например /getfruit100.
		if len(cm.Text) > 4 && cm.Text[:4] == "/get" {
			rest := cm.Text[4:]
			// Отделяем хвостовые цифры (количество).
			i := len(rest)
			for i > 0 && rest[i-1] >= '0' && rest[i-1] <= '9' {
				i--
			}
			name := rest[:i]
			qtyStr := rest[i:]
			if name == "" || qtyStr == "" {
				c.sendEnvelope(protocol.TypeChat, protocol.ChatMessage{
					From: "server", Text: "usage: /get<item><qty>", TS: time.Now().UnixMilli(),
				})
				break
			}
			qty := 0
			for _, ch := range qtyStr {
				qty = qty*10 + int(ch-'0')
			}
			if qty > 100000 {
				qty = 100000
			}
			c.mu.Lock()
			c.inventory[name] += qty
			inv := make(map[string]int, len(c.inventory))
			for k, v := range c.inventory {
				inv[k] = v
			}
			c.mu.Unlock()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("cheat: get", "item", name, "qty", qty)
			break
		}
		cm.From = c.Nick
		cm.TS = time.Now().UnixMilli()
		s.broadcast(protocol.TypeChat, cm)
		s.metrics.ChatMessages.Inc()
	case protocol.TypePing:
		var p protocol.Ping
		_ = env.Decode(&p)
		c.sendEnvelope(protocol.TypePong, protocol.Pong{
			Sent:     p.Sent,
			ServerTS: time.Now().UnixMilli(),
		})
	case protocol.TypeEatFruit:
		if c.consumeItem("fruit") {
			c.addHunger(20)
			inv := c.inventorySnapshot()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("ate fruit", "hunger", c.Hunger())
		}
	case protocol.TypePickupItem:
		var p protocol.PickupItem
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePickup(c, p.ResourceID)
	case protocol.TypeThrowSpear:
		var p protocol.ThrowSpear
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleThrowSpear(c, p.Dir)
	case protocol.TypeCraftItem:
		var p protocol.CraftItem
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleCraft(c, p.Recipe)
	case protocol.TypeHitMammoth:
		var p protocol.HitMammoth
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleHitMammoth(c, p.MammothID)
	case protocol.TypePlantSeed:
		var p protocol.PlantSeed
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlantSeed(c, p)
	case protocol.TypeWaterPlant:
		var p protocol.WaterPlant
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleWaterPlant(c, p.ResourceID)
	case protocol.TypeTakeWater:
		var p protocol.TakeWater
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleTakeWater(c, p.WellID)
	case protocol.TypeTameMammoth:
		var p protocol.TameMammoth
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleTameMammoth(c, p.MammothID)
	case protocol.TypeSelectItem:
		var p protocol.SelectItem
		if err := env.Decode(&p); err != nil {
			return
		}
		c.mu.Lock()
		c.heldItem = p.Item
		c.mu.Unlock()
		c.log.Info("select_item received", "item", p.Item)
	case protocol.TypeLeashMammoth:
		var p protocol.LeashMammoth
		if err := env.Decode(&p); err != nil {
			return
		}
		c.log.Info("leash packet received", "id", p.MammothID)
		s.handleLeashMammoth(c, p.MammothID)
	case protocol.TypePlaceHouse:
		var p protocol.PlaceHouse
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlaceHouse(c, p)
	case protocol.TypeToggleDoor:
		var p protocol.ToggleDoor
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleToggleDoor(c, p.HouseID)
	case protocol.TypeSaddleMammoth:
		var p protocol.SaddleMammoth
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleSaddleMammoth(c, p.MammothID)
	case protocol.TypeRideMammoth:
		var p protocol.RideMammoth
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleRideMammoth(c, p.MammothID)
	case protocol.TypePlaceBoat:
		var p protocol.PlaceBoat
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlaceBoat(c, p)
	case protocol.TypeEnterBoat:
		var p protocol.EnterBoat
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleEnterBoat(c, p.BoatID)
	case protocol.TypeHitMob:
		var p protocol.HitMob
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleHitMob(c, p.MobID)
	case protocol.TypeAcceptContract:
		var p protocol.AcceptContract
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleAcceptContract(c, p.MobID, p.ContractID)
	case protocol.TypePlaceSolar:
		var p protocol.PlaceSolar
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlaceSolar(c, p)
	case protocol.TypePlaceBattery:
		var p protocol.PlaceBattery
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlaceBattery(c, p)
	case protocol.TypePlaceFactory:
		var p protocol.PlaceFactory
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlaceFactory(c, p)
	case protocol.TypeOpenFactory:
		var p protocol.OpenFactory
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleOpenFactory(c, p.FactoryID)
	case protocol.TypeCraftFactory:
		var p protocol.CraftFactory
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleCraftFactory(c, p.FactoryID, p.Recipe)
	}
}

func (s *Server) handlePickup(c *Client, resourceID string) {
	ps := c.State()

	s.resourcesMu.Lock()
	r, ok := s.resources[resourceID]
	if !ok || r.Type == "seed" {
		s.resourcesMu.Unlock()
		return
	}
	dx := float64(r.X - ps.X)
	dy := float64(r.Y - ps.Y)
	dz := float64(r.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > pickupRadius*pickupRadius {
		s.resourcesMu.Unlock()
		return
	}
	delete(s.resources, resourceID)
	s.resourcesMu.Unlock()

	inv := c.addItem(r.Type)
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("item picked up", "item", r.Type)
}

func (s *Server) broadcast(t protocol.Type, data any) {
	env, err := protocol.NewEnvelope(t, data)
	if err != nil {
		return
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.clients {
		c.enqueue(raw)
	}
}

func (s *Server) tickLoop() {
	defer s.wg.Done()
	interval := time.Second / time.Duration(s.tickRate)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			s.tick++
			s.metrics.TicksTotal.Inc()
			s.tickHunger(1.0 / float64(s.tickRate))
			s.tickMammoths(float32(1.0 / float64(s.tickRate)))
			s.tickBoats()
			s.tickMobs(float32(1.0 / float64(s.tickRate)))
			s.tickProjectiles(float32(1.0 / float64(s.tickRate)))
			s.tickBreeding()
			s.tickResources()
			s.tickEnergy(float32(1.0 / float64(s.tickRate)))
			s.broadcastSnapshot()
		case <-s.done:
			return
		}
	}
}

func (s *Server) tickHunger(dt float64) {
	const hungerRate = 0.333
	const autoEatThreshold = 50.0
	const fruitValue = 20.0

	dec := float32(hungerRate * dt)

	s.mu.RLock()
	clients := make([]*Client, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.RUnlock()

	for _, c := range clients {
		c.addHunger(-dec)

		// Автосъедание: fruit (20), затем meat (30)
		for c.Hunger() < autoEatThreshold {
			eaten := ""
			if c.consumeItem("fruit") {
				c.addHunger(20)
				eaten = "fruit"
			} else if c.consumeItem("meat") {
				c.addHunger(30)
				eaten = "meat"
			}
			if eaten == "" {
				break
			}
			inv := c.inventorySnapshot()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("auto-ate", "item", eaten, "hunger", c.Hunger())
		}
	}
}

func (s *Server) broadcastSnapshot() {
	s.mu.RLock()
	players := make([]protocol.PlayerState, 0, len(s.clients))
	for _, c := range s.clients {
		ps := c.State()
		ps.Hunger = c.Hunger()
		players = append(players, ps)
	}
	s.mu.RUnlock()

	s.resourcesMu.RLock()
	resources := make([]protocol.Resource, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, r)
	}
	s.resourcesMu.RUnlock()

	s.wellsMu.RLock()
	wells := make([]protocol.Well, 0, len(s.wells))
	for _, w := range s.wells {
		wells = append(wells, *w)
	}
	s.wellsMu.RUnlock()

	s.projMu.RLock()
	projs := make([]protocol.MobProjectile, 0, len(s.projectiles))
	for _, p := range s.projectiles {
		projs = append(projs, protocol.MobProjectile{
			ID: p.ID,
			X:  p.Pos.X,
			Y:  p.Pos.Y,
			Z:  p.Pos.Z,
			DX: p.Dir.X,
			DY: p.Dir.Y,
			DZ: p.Dir.Z,
		})
	}
	s.projMu.RUnlock()

	s.mobsMu.RLock()
	mobs := make([]protocol.Mob, 0, len(s.mobs))
	for _, m := range s.mobs {
		mobs = append(mobs, protocol.Mob{
			ID:       m.ID,
			X:        m.Pos.X,
			Y:        m.Pos.Y,
			Z:        m.Pos.Z,
			HP:       m.HP,
			Kind:     m.Kind,
			Contract: m.Contract,
			OwnerID:  m.OwnerID,
		})
	}
	s.mobsMu.RUnlock()

	s.boatsMu.RLock()
	boats := make([]protocol.Boat, 0, len(s.boats))
	for _, b := range s.boats {
		boats = append(boats, protocol.Boat{
			ID:      b.ID,
			X:       b.Pos.X,
			Y:       b.Pos.Y,
			Z:       b.Pos.Z,
			Yaw:     b.Yaw,
			RiderID: b.RiderID,
		})
	}
	s.boatsMu.RUnlock()

	s.housesMu.RLock()
	houses := make([]protocol.House, 0, len(s.houses))
	for _, h := range s.houses {
		houses = append(houses, protocol.House{
			ID:       h.ID,
			X:        h.Pos.X,
			Y:        h.Pos.Y,
			Z:        h.Pos.Z,
			Yaw:      h.Yaw,
			DoorOpen: h.DoorOpen,
		})
	}
	s.housesMu.RUnlock()

	s.solarMu.RLock()
	solars := make([]protocol.Solar, 0, len(s.solar))
	for _, sl := range s.solar {
		solars = append(solars, protocol.Solar{
			ID: sl.ID, X: sl.Pos.X, Y: sl.Pos.Y, Z: sl.Pos.Z, Yaw: sl.Yaw,
		})
	}
	s.solarMu.RUnlock()

	s.batteriesMu.RLock()
	batteries := make([]protocol.Battery, 0, len(s.batteries))
	for _, b := range s.batteries {
		batteries = append(batteries, protocol.Battery{
			ID: b.ID, X: b.Pos.X, Y: b.Pos.Y, Z: b.Pos.Z, Yaw: b.Yaw,
			Energy: b.Energy, MaxEnergy: b.MaxEnergy,
		})
	}
	s.batteriesMu.RUnlock()

	s.factoriesMu.RLock()
	factories := make([]protocol.Factory, 0, len(s.factories))
	for _, f := range s.factories {
		factories = append(factories, protocol.Factory{
			ID: f.ID, X: f.Pos.X, Y: f.Pos.Y, Z: f.Pos.Z, Yaw: f.Yaw,
			Crafting: f.Crafting, Progress: f.Progress,
		})
	}
	s.factoriesMu.RUnlock()

	s.mammothsMu.RLock()
	mammoths := make([]protocol.Mammoth, 0, len(s.mammoths))
	for _, m := range s.mammoths {
		mammoths = append(mammoths, protocol.Mammoth{
			ID:        m.ID,
			X:         m.Pos.X,
			Y:         m.Pos.Y,
			Z:         m.Pos.Z,
			HP:        m.HP,
			Tamed:     m.Tamed,
			Sex:       m.Sex,
			FedCount:  m.FedCount,
			Baby:      m.Baby,
			LeashedTo: m.LeashedTo,
			Saddle:    m.Saddle,
			RiderID:   m.RiderID,
		})
	}
	s.mammothsMu.RUnlock()

	env, err := protocol.NewEnvelope(protocol.TypeSnapshot, protocol.Snapshot{
		Tick:        s.tick,
		Players:     players,
		Resources:   resources,
		Mammoths:    mammoths,
		Wells:       wells,
		Houses:      houses,
		Boats:       boats,
		Mobs:        mobs,
		Projectiles: projs,
		Solar:       solars,
		Batteries:   batteries,
		Factories:   factories,
	})
	if err != nil {
		return
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.clients {
		c.enqueue(raw)
	}
}

func readEnvelope(r io.Reader) (*protocol.Envelope, error) {
	payload, err := protocol.ReadFrame(r)
	if err != nil {
		return nil, err
	}
	var env protocol.Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
