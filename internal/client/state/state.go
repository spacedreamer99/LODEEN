package state

type Mode int

const (
	ModeMenu Mode = iota
	ModePlaying
	ModePaused
)
