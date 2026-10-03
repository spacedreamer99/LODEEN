package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGet_ReturnsCurrentVars(t *testing.T) {
	info := Get()
	if info.Version != Version {
		t.Fatalf("Version = %q, want %q", info.Version, Version)
	}
	if info.Commit != Commit {
		t.Fatalf("Commit = %q, want %q", info.Commit, Commit)
	}
	if info.Date != Date {
		t.Fatalf("Date = %q, want %q", info.Date, Date)
	}
	if info.GoVersion != runtime.Version() {
		t.Fatalf("GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
}

func TestGet_GoVersionNonEmpty(t *testing.T) {
	if Get().GoVersion == "" {
		t.Fatalf("GoVersion should never be empty")
	}
}

func TestDefaults_NonEmpty(t *testing.T) {
	// Без ldflags-инъекции переменные должны быть заданы дефолтами,
	// а не пустыми строками — иначе логи и /version будут бесполезны.
	if Version == "" {
		t.Fatalf("Version default is empty")
	}
	if Commit == "" {
		t.Fatalf("Commit default is empty")
	}
	if Date == "" {
		t.Fatalf("Date default is empty")
	}
}

func TestString_ContainsVersionAndCommit(t *testing.T) {
	s := String()
	if !strings.Contains(s, Version) {
		t.Fatalf("String() = %q, missing Version %q", s, Version)
	}
	if !strings.Contains(s, Commit) {
		t.Fatalf("String() = %q, missing Commit %q", s, Commit)
	}
	if !strings.Contains(s, Date) {
		t.Fatalf("String() = %q, missing Date %q", s, Date)
	}
}

func TestString_NonEmpty(t *testing.T) {
	if String() == "" {
		t.Fatalf("String() is empty")
	}
}
