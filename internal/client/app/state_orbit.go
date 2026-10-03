package app

// OrbitState — состояние 2D-карты орбиты (клавиша M) и экранных
// координат небесных тел, которые переиспользуются в HUD и orbitmap.
type OrbitState struct {
	// Открыта ли карта.
	showOrbitMap bool

	// Управление камерой орбиты.
	orbitAzimuth     float32
	orbitElevation   float32
	orbitDistance    float32
	orbitInit        bool
	orbitFocus       string // "" | "earth" | "sun" | "rocket"
	orbitDragged     bool
	orbitMouseStartX float32
	orbitMouseStartY float32

	// Проекции на экран (переиспользуются в HUD/orbitmap).
	earthScrX   float32
	earthScrY   float32
	earthScrR   float32
	sunScrX     float32
	sunScrY     float32
	star2ScrX   float32
	star2ScrY   float32
	planet2ScrX float32
	planet2ScrY float32
	rocketScrX  float32
	rocketScrY  float32
}
