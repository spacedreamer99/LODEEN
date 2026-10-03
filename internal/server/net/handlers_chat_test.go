package net

import "testing"

// TestParseGetArg проверяет парсер команды /get<item><qty>.
// Чистая функция, без сервера/клиента.
func TestParseGetArg(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantItem string
		wantQty  int
		wantOK   bool
	}{
		{"simple", "stone10", "stone", 10, true},
		{"single_digit", "wood1", "wood", 1, true},
		{"zero_qty", "a0", "a", 0, true},
		{"multi_digit", "apple123", "apple", 123, true},
		{"clamp_huge", "x999999999999", "x", 100000, true},
		{"underscore_in_name", "my_item5", "my_item", 5, true},
		{"no_qty", "stone", "", 0, false},
		{"no_name", "123", "", 0, false},
		{"empty", "", "", 0, false},
		{"only_letters", "abc", "", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotItem, gotQty, gotOK := parseGetArg(tc.in)
			if gotOK != tc.wantOK {
				t.Fatalf("ok = %v, want %v (in=%q)", gotOK, tc.wantOK, tc.in)
			}
			if gotItem != tc.wantItem {
				t.Fatalf("item = %q, want %q (in=%q)", gotItem, tc.wantItem, tc.in)
			}
			if gotQty != tc.wantQty {
				t.Fatalf("qty = %d, want %d (in=%q)", gotQty, tc.wantQty, tc.in)
			}
		})
	}
}

// TestParseGetArg_ClampExactlyAtLimit — на границе 100000 не должен клампить в 0.
func TestParseGetArg_ClampExactlyAtLimit(t *testing.T) {
	item, qty, ok := parseGetArg("x100000")
	if !ok || item != "x" || qty != 100000 {
		t.Fatalf("got (%q,%d,%v), want (\"x\",100000,true)", item, qty, ok)
	}
}

// TestParseGetArg_OverLimit — за границей клампит до 100000.
func TestParseGetArg_OverLimit(t *testing.T) {
	item, qty, ok := parseGetArg("x100001")
	if !ok || item != "x" || qty != 100000 {
		t.Fatalf("got (%q,%d,%v), want (\"x\",100000,true)", item, qty, ok)
	}
}
