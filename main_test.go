package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestAllowedPrinterHost(t *testing.T) {
	allow := []string{"192.168.1.79", "10.0.0.4", "127.0.0.1", "localhost", "printer.local", "172.16.1.2"}
	deny := []string{"8.8.8.8", "1.1.1.1", "169.254.169.254", "metadata.google.internal", "app.3djobdesk.com"}
	for _, host := range allow {
		if !allowedPrinterHost(host) {
			t.Fatalf("expected allow %s", host)
		}
	}
	for _, host := range deny {
		if allowedPrinterHost(host) {
			t.Fatalf("expected deny %s", host)
		}
	}
}

func TestNormalizeOrigin(t *testing.T) {
	ok, err := normalizeOrigin("https://app.3djobdesk.com")
	if err != nil || ok != "https://app.3djobdesk.com" {
		t.Fatalf("https origin: %s %v", ok, err)
	}
	if _, err := normalizeOrigin("http://evil.example"); err == nil {
		t.Fatal("expected http public origin to fail")
	}
	if _, err := normalizeOrigin("http://127.0.0.1:43181"); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultOrigin(t *testing.T) {
	if defaultOrigin != "https://app.3djobdesk.com" {
		t.Fatalf("website must stay %s", defaultOrigin)
	}
}

func TestClaimPairing(t *testing.T) {
	configPathOverride = filepath.Join(t.TempDir(), "bridge.json")
	t.Cleanup(func() { configPathOverride = "" })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/printers/bridge/claim" {
			t.Fatalf("path %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]string
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		if body["code"] != "ABCD-EFGH" {
			t.Fatalf("code %s", body["code"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"deviceId":"d1","deviceSecret":"s1","deskName":"Shop"}`)
	}))
	defer srv.Close()

	cfg, err := claimPairing(srv.URL, "ABCD-EFGH", "Test PC")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DeviceID != "d1" || cfg.DeskName != "Shop" {
		t.Fatalf("%+v", cfg)
	}
	loaded, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DeviceSecret != "s1" {
		t.Fatalf("%+v", loaded)
	}
}

func TestClaimPairingRequiresCode(t *testing.T) {
	if _, err := claimPairing(defaultOrigin, "  ", "Shop"); err == nil {
		t.Fatal("expected missing code to fail")
	}
}
