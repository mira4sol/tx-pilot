package config_test

import (
	"os"
	"testing"

	"github.com/mira4sol/aegis/internal/config"
	"github.com/mira4sol/aegis/pkg/aegis"
)

func TestValidatePolicyMode(t *testing.T) {
	cfg := &config.Config{
		SolanaRPCURL: "http://localhost",
		YellowstoneGRPCURL: "localhost:443",
		YellowstoneGRPCToken: "token",
		StorageDSN: "postgres://",
		OpenAIAPIKey: "key",
		KeypairPath: "/tmp/key",
		PolicyMode: aegis.ModeSafe,
	}
	if err := config.Validate(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("SOLANA_RPC_URL", "http://rpc")
	t.Setenv("YELLOWSTONE_GRPC_URL", "grpc:443")
	t.Setenv("YELLOWSTONE_GRPC_TOKEN", "tok")
	t.Setenv("DATABASE_URL", "postgres://localhost/aegis_v2")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("AEGIS_KEYPAIR_PATH", "/tmp/key.json")
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PolicyMode != aegis.ModeSafe {
		t.Fatalf("policy %s", cfg.PolicyMode)
	}
	_ = os.Unsetenv("SOLANA_RPC_URL")
}
