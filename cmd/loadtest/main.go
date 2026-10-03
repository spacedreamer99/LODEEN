// Command loadtest is a headless load generator for a LODEEN world server.
//
// It spawns N bot clients that connect over TCP, complete the Hello/Welcome
// handshake, then send PlayerState at a fixed rate and consume snapshots.
// No graphics, no raylib — pure net load.
//
// Usage:
//
//	./bin/lodeen-loadtest -n 32 -addr 127.0.0.1:7777 -rate 20 -duration 60s
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

type stats struct {
	snapshots atomic.Uint64
	bytes     atomic.Uint64
	errors    atomic.Uint64
}

func main() {
	var (
		n        = flag.Int("n", 10, "number of bots")
		addr     = flag.String("addr", "127.0.0.1:7777", "server TCP address")
		rate     = flag.Int("rate", 20, "PlayerState messages per second")
		duration = flag.Duration("duration", 60*time.Second, "test duration (0 = until Ctrl+C)")
		ramp     = flag.Duration("ramp", 100*time.Millisecond, "delay between bot startups")
	)
	flag.Parse()

	fmt.Printf("loadtest: bots=%d addr=%s rate=%dHz duration=%s ramp=%s\n",
		*n, *addr, *rate, *duration, *ramp)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if *duration > 0 {
		var c context.CancelFunc
		ctx, c = context.WithTimeout(ctx, *duration)
		defer c()
	}

	var st stats
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < *n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			bot(ctx, id, *addr, *rate, &st)
		}(i)
		time.Sleep(*ramp)
	}

	// Reporter: prints throughput every 5 seconds.
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		lastSnap, lastBytes := uint64(0), uint64(0)
		for {
			select {
			case <-done:
				return
			case <-t.C:
				curSnap := st.snapshots.Load()
				curBytes := st.bytes.Load()
				elapsed := time.Since(start).Seconds()
				fmt.Printf("[%5.1fs] snapshots=%d (%.0f/s) bytes=%d (%.1f KB/s) errors=%d\n",
					elapsed, curSnap,
					float64(curSnap-lastSnap)/5.0,
					curBytes,
					float64(curBytes-lastBytes)/5.0/1024.0,
					st.errors.Load())
				lastSnap, lastBytes = curSnap, curBytes
			}
		}
	}()

	wg.Wait()
	close(done)

	elapsed := time.Since(start)
	totalSnap := st.snapshots.Load()
	avgPerSnap := float64(st.bytes.Load()) / float64(totalSnap+1)

	fmt.Printf("\n=== done in %s ===\n", elapsed.Round(time.Millisecond))
	fmt.Printf("bots:           %d\n", *n)
	fmt.Printf("snapshots:      %d (avg %.0f/s)\n", totalSnap, float64(totalSnap)/elapsed.Seconds())
	fmt.Printf("bytes received: %d (avg %.1f B/snapshot, %.1f KB/s)\n",
		st.bytes.Load(), avgPerSnap, float64(st.bytes.Load())/elapsed.Seconds()/1024)
	fmt.Printf("errors:         %d\n", st.errors.Load())
}

// bot runs one load-generating client until ctx is cancelled.
func bot(ctx context.Context, id int, addr string, rate int, st *stats) {
	nick := fmt.Sprintf("bot-%03d", id)
	rng := rand.New(rand.NewSource(int64(id)*7919 + time.Now().UnixNano()))

	// Pick a random spawn direction on the sphere.
	theta := rng.Float64() * 2 * math.Pi
	phi := math.Acos(2*rng.Float64() - 1)
	nx := float32(math.Sin(phi) * math.Cos(theta))
	ny := float32(math.Cos(phi))
	nz := float32(math.Sin(phi) * math.Sin(theta))
	h := protocol.TerrainHeight(nx, ny, nz) + protocol.PlayerHeight
	baseX := nx * h
	baseY := ny * h
	baseZ := nz * h
	yaw := float32(rng.Float64() * 2 * math.Pi)

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		st.errors.Add(1)
		return
	}
	defer conn.Close()

	// --- Handshake ---
	helloEnv, _ := protocol.NewEnvelope(protocol.TypeHello, protocol.Hello{
		Nick: nick, Version: "loadtest",
	})
	rawHello, _ := json.Marshal(helloEnv)
	if err := protocol.WriteFrame(conn, rawHello); err != nil {
		st.errors.Add(1)
		return
	}

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	welcomeRaw, err := protocol.ReadFrame(conn)
	if err != nil {
		st.errors.Add(1)
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	var welcomeEnv protocol.Envelope
	if err := json.Unmarshal(welcomeRaw, &welcomeEnv); err != nil {
		st.errors.Add(1)
		return
	}
	if welcomeEnv.Type != protocol.TypeWelcome {
		st.errors.Add(1)
		return
	}
	var welcome protocol.Welcome
	_ = welcomeEnv.Decode(&welcome)
	playerID := welcome.PlayerID

	// --- Reader: consume snapshots ---
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			frame, err := protocol.ReadFrame(conn)
			if err != nil {
				return
			}
			st.snapshots.Add(1)
			st.bytes.Add(uint64(len(frame)))
		}
	}()

	// --- Sender: PlayerState at fixed rate + Ping every second ---
	sendTick := time.NewTicker(time.Second / time.Duration(rate))
	defer sendTick.Stop()
	pingTick := time.NewTicker(time.Second)
	defer pingTick.Stop()

	angle := 0.0
	for {
		select {
		case <-ctx.Done():
			return
		case <-sendTick.C:
			angle += 0.01
			c := float32(math.Cos(angle))
			s := float32(math.Sin(angle))
			// Rotate base position around Y-axis (bob in place).
			px := baseX*c + baseZ*s
			pz := -baseX*s + baseZ*c

			ps := protocol.PlayerState{
				ID:     playerID,
				Nick:   nick,
				X:      px,
				Y:      baseY,
				Z:      pz,
				Yaw:    yaw,
				Pitch:  0,
				Hunger: 100,
				HP:     100,
			}
			env, _ := protocol.NewEnvelope(protocol.TypeState, ps)
			raw, _ := json.Marshal(env)
			_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if err := protocol.WriteFrame(conn, raw); err != nil {
				st.errors.Add(1)
				return
			}
		case <-pingTick.C:
			env, _ := protocol.NewEnvelope(protocol.TypePing, protocol.Ping{
				Sent: time.Now().UnixMilli(),
			})
			raw, _ := json.Marshal(env)
			_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			_ = protocol.WriteFrame(conn, raw)
		}
	}
}
