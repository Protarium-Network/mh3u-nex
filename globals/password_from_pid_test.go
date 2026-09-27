package globals

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/plogger-go"
)

func TestPasswordFromPIDDerivedIsStableAndKeyed(t *testing.T) {
	LocalAuthMode = false
	NEXPasswordSecret = make([]byte, 32)
	for i := range NEXPasswordSecret {
		NEXPasswordSecret[i] = byte(i)
	}

	a, code := PasswordFromPID(types.NewPID(1234))
	if code != 0 || a == "" {
		t.Fatalf("expected a password, got code=%d value=%q", code, a)
	}
	if b, _ := PasswordFromPID(types.NewPID(1234)); b != a {
		t.Fatalf("derivation not stable: %q != %q", a, b)
	}
	if c, _ := PasswordFromPID(types.NewPID(1235)); c == a {
		t.Fatal("different PIDs produced the same password")
	}

	NEXPasswordSecret = make([]byte, 16) // too short
	if _, code := PasswordFromPID(types.NewPID(1234)); code == 0 {
		t.Fatal("expected failure with a short secret")
	}
}

func TestPasswordFromPIDLocalReadsSettings(t *testing.T) {
	Logger = plogger.NewLogger()
	LocalAuthMode = true

	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"accounts":[{"username":"42","password":"hunter2"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PN_MH3U_SETTINGS_PATH", path)

	if got, code := PasswordFromPID(types.NewPID(42)); code != 0 || got != "hunter2" {
		t.Fatalf("got (%q, %d), want (\"hunter2\", 0)", got, code)
	}
	if _, code := PasswordFromPID(types.NewPID(99)); code == 0 {
		t.Fatal("expected failure for a PID missing from settings.json")
	}
}
