package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsRootConfigEnvWithPrefixedKeys(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	})

	if err := os.WriteFile(
		filepath.Join(tmp, "config.env"),
		[]byte("SNXMCP_SERVER_ADDRESS=127.0.0.1:9999\nSNXMCP_AGENT_BROKER_ENABLED=false\n"),
		0o600,
	); err != nil {
		t.Fatalf("write config.env: %v", err)
	}

	v, err := Load("SNXMCP")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got := v.GetString("server_address"); got != "127.0.0.1:9999" {
		t.Fatalf("expected root config.env server_address, got %q", got)
	}
	if got := v.GetBool("agent_broker_enabled"); got {
		t.Fatal("expected root config.env broker disable")
	}
}

func TestLoadEnvOverridesRootConfigEnv(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	})

	if err := os.WriteFile(
		filepath.Join(tmp, "config.env"),
		[]byte("SNXMCP_SERVER_ADDRESS=127.0.0.1:9999\n"),
		0o600,
	); err != nil {
		t.Fatalf("write config.env: %v", err)
	}
	t.Setenv("SNXMCP_SERVER_ADDRESS", "127.0.0.1:8888")

	v, err := Load("SNXMCP")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got := v.GetString("server_address"); got != "127.0.0.1:8888" {
		t.Fatalf("expected env override, got %q", got)
	}
}
