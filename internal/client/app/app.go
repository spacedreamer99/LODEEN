package app

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/chat"
	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/input"
	clientnet "github.com/spacedreamer99/lodeen/internal/client/net"
	"github.com/spacedreamer99/lodeen/internal/client/render"
	"github.com/spacedreamer99/lodeen/internal/client/state"
	"github.com/spacedreamer99/lodeen/internal/shared/config"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

const (
	screenW = 1280
	screenH = 720
)

type projectile struct {
	pos   rl.Vector3
	dir   rl.Vector3
	spawn time.Time
}

type App struct {
	cfg *config.Config
	log *slog.Logger

	mode state.Mode

	nc     *clientnet.Client
	flight *input.FlightController
	scene  *render.Scene
	chat   *chat.Chat

	uiTarget  rl.RenderTexture2D
	uiTargetW int32
	uiTargetH int32

	quit        bool
	projectiles []projectile
	startAt     time.Time

	lastSentHeld string
	ridingID     string
	hp           int
	hpReceived   bool
	boatID       string

	// flightLocked — при открытом UI игрок жёстко привязан к Земле.
	// Позволяет не «улетать» когда updatePlaying делает early-return.
	flightLocked       bool
	flightLockedRelPos protocol.Vector3
	flightLockedRelVel protocol.Vector3
	rocketID           string
	rocketBoardedAt    time.Time
	hudRocket          protocol.Rocket
	showOrbitMap       bool
	orbitAzimuth       float32
	orbitElevation     float32
	orbitDistance      float32
	orbitInit          bool
	orbitFocus         string // "" | "earth" | "sun" | "rocket"
	orbitDragged       bool
	orbitMouseStartX   float32
	orbitMouseStartY   float32
	earthScrX          float32
	earthScrY          float32
	earthScrR          float32
	sunScrX            float32
	sunScrY            float32
	star2ScrX          float32
	star2ScrY          float32
	planet2ScrX        float32
	planet2ScrY        float32
	rocketScrX         float32
	rocketScrY         float32
	autoPilot          string
	rocketNoseX        float32
	rocketNoseY        float32
	rocketNoseZ        float32
	rocketNoseInit     bool

	pausedRelPos protocol.Vector3
	pausedRelVel protocol.Vector3

	unfocusFreeze bool
	unfocusRelPos protocol.Vector3
	unfocusRelVel protocol.Vector3

	diag   DiagState
	ui     UIState
	world  WorldState
	camera CameraState
}

func New(cfg *config.Config, log *slog.Logger) *App {
	return &App{
		hp:   100,
		cfg:  cfg,
		log:  log,
		mode: state.ModeMenu,
		ui: UIState{
			menuNick: cfg.Client.Nick,
			menuAddr: cfg.Client.StartAddr,
		},
	}
}

func (a *App) Run() error {
	rl.SetConfigFlags(rl.FlagVsyncHint | rl.FlagWindowResizable)
	rl.InitWindow(screenW, screenH, "LODEEN")
	defer rl.CloseWindow()
	rl.SetTargetFPS(0)
	rl.SetExitKey(rl.KeyNull)

	a.resizeUITarget()
	a.startAt = time.Now()
	fonts.Load(28)
	if os.Getenv("LODEEN_FONT_DEBUG") == "1" {
		fonts.DebugDump()
	}

	a.scene = render.NewScene(a.cfg.Client.PlanetModel)
	defer a.scene.Unload()
	if a.scene.HasPlanet() {
		a.log.Info("planet model loaded", "path", a.cfg.Client.PlanetModel)
	} else {
		a.log.Warn("planet model not loaded, using fallback sphere",
			"path", a.cfg.Client.PlanetModel)
	}

	a.nc = clientnet.New(a.log)
	a.chat = chat.New()

	a.camera.camera = rl.Camera3D{
		Position:   rl.NewVector3(0, 5, 40),
		Target:     rl.NewVector3(0, 0, 0),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       70,
		Projection: rl.CameraPerspective,
	}

	frameTimes := make([]float32, 0, 60)
	for !rl.WindowShouldClose() && !a.quit {
		dt := rl.GetFrameTime()
		frameTimes = append(frameTimes, dt)
		if len(frameTimes) >= 60 {
			var mn, mx, sum float32
			mn = frameTimes[0]
			for _, t := range frameTimes {
				if t < mn {
					mn = t
				}
				if t > mx {
					mx = t
				}
				sum += t
			}
			avg := sum / 60
			a.log.Info("frame stats",
				"min", fmt.Sprintf("%.4f", mn),
				"max", fmt.Sprintf("%.4f", mx),
				"avg", fmt.Sprintf("%.4f", avg),
				"jitter", fmt.Sprintf("%.4f", mx-mn))
			frameTimes = frameTimes[:0]
		}
		a.update(dt)
		a.draw()
	}
	a.setCursorCaptured(false)
	a.nc.Disconnect()
	return nil
}

func (a *App) setCursorCaptured(c bool) {
	if a.ui.cursorCaptured == c {
		return
	}
	a.ui.cursorCaptured = c
	if c {
		rl.DisableCursor()
		midX := rl.GetScreenWidth() / 2
		midY := rl.GetScreenHeight() / 2
		rl.SetMousePosition(midX, midY)
	} else {
		rl.EnableCursor()
		rl.ShowCursor()
	}
}

// resizeUITarget пересоздаёт UI-буфер под текущий размер окна.

// updateDead обрабатывает горячие клавиши на экране смерти.
// factoryRecipeUI — рецепты завода для UI.
type factoryRecipeUI struct {
	id     string
	name   string
	need   map[string]int
	energy int
}

var factoryRecipeList = []factoryRecipeUI{
	{id: "steel", name: "Steel", need: map[string]int{"ore": 5}, energy: 10},
	{id: "gear", name: "Gear", need: map[string]int{"stone": 2, "wood": 2}, energy: 20},
	{id: "circuit", name: "Circuit", need: map[string]int{"ore": 3, "liana": 1}, energy: 50},
	{id: "drone", name: "Drone", need: map[string]int{"gear": 1, "circuit": 1}, energy: 100},
	{id: "rocket", name: "Rocket", need: map[string]int{"circuit": 3, "steel": 5, "gear": 2}, energy: 500},
}

// itoa — простой int→string (без зависимостей).

// updatePilotedRocket — если игрок в ракете, обновляет её состояние на клиенте.
// Возвращает true если мы по-прежнему в ракете.

// drawRocketHUD — оверлей с орбитальной информацией, пока сидим в ракете.
// predictTrajectory — быстрая симуляция орбиты на N шагов вперёд.
// Копирует серверную физику (без атмосферы для простоты).

// drawOrbitMap — 2D проекция орбиты сверху.
// updateOrbitMapInput — управление орбитальной камерой (drag + zoom).

// drawOrbitMap — 3D вид на орбиту вокруг планеты.
// rotateAroundAxis — поворот вектора вокруг оси (формула Родрига).

// drawNavBall — простой 2D навбол: показывает направление носа ракеты
// и целевые векторы (prograde/retrograde/radial/normal).

// respawn переподключается и возвращает в игру с полным HP.

// returnToMenu возвращает в главное меню.

// drawDead рисует экран смерти с двумя кнопками.

// drawHotbar — первая строка инвентаря внизу по центру.

// seedInSight возвращает ID ближайшего seed в радиусе 5 юнитов
// и в конусе ~53° по направлению взгляда.
// wellInSight возвращает ID ближайшего источника пресной воды
// в радиусе 6 юнитов и в конусе взгляда.

// mammothInReach возвращает ID ближайшего мамонта в радиусе 5 юнитов
// и в конусе взгляда.
// houseInSight возвращает ID ближайшего дома в радиусе 6 юнитов.
// pinkMobInSight — ближайший розовый моб в конусе взгляда.
// factoryInSight — ближайший завод в радиусе 6 юнитов и в конусе взгляда.
// rocketInSight — ближайшая ракета в радиусе 8 юнитов и в конусе взгляда.

// placeForward возвращает точку перед игроком и её yaw.

// drawInventory — сетка 16×16 с подсветкой первой строки и tooltip.

// itemColor возвращает цвет иконки для типа ресурса.

// itemName — человекочитаемое имя.

// craftRecipes — клиентский список рецептов для UI.
// Сервер валидирует независимо, здесь — только отображение.
type craftRecipe struct {
	id   string
	name string
	out  string
	req  map[string]int
}

var craftRecipes = []craftRecipe{
	{id: "spear", name: "Spear", out: "spear", req: map[string]int{"stone": 2, "wood": 1}},
	{id: "torch", name: "Torch", out: "torch", req: map[string]int{"stone": 1, "wood": 1}},
	{id: "leash", name: "Leash", out: "leash", req: map[string]int{"liana": 2}},
	{id: "house", name: "House", out: "house", req: map[string]int{"wood": 50}},
	{id: "saddle", name: "Saddle", out: "saddle", req: map[string]int{"liana": 4}},
	{id: "boat", name: "Boat", out: "boat", req: map[string]int{"wood": 20}},
	{id: "solar", name: "Solar Panel", out: "solar", req: map[string]int{"ore": 10, "stone": 5}},
	{id: "battery", name: "Battery", out: "battery", req: map[string]int{"ore": 15, "stone": 10}},
	{id: "factory", name: "Factory", out: "factory", req: map[string]int{"ore": 30, "wood": 20, "stone": 20}},
}

// drawCraft — окно крафта со списком рецептов.

// drawHeldItem — предмет в руках: куб перед камерой.
// Работает только в Survival, если в selectedSlot есть предмет.

// heldItem возвращает тип предмета в selectedSlot (пустая строка — ничего).

// syncInvSlots — разложить новые предметы из серверного инвентаря по слотам.
// Уже занятые слоты не трогает, освобождает слоты при исчезновении предмета.

// updateProjectiles — локальная симуляция снарядов с детекцией коллизии.

// segSphereHit — пересечение отрезка [a,b] со сферой (c, r).

// drawProjectiles — рисует локальные снаряды жёлтыми сферами.

// drawMiniStatus — компактный HUD: тело, состояние (ON GROUND/AIRBORNE), высота, скорость.
