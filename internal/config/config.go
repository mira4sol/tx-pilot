package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mira4sol/aegis/pkg/aegis"
)

type Config struct {
	Env                  string
	Cluster              string
	HTTPAddr             string
	PolicyMode           aegis.PolicyMode
	PublicAPIBaseURL     string
	SolanaRPCURL         string
	SolanaWSURL          string
	SolanaRPCAPIKey      string
	YellowstoneGRPCURL   string
	YellowstoneGRPCToken string
	SolInfraRESTAPIURL   string
	SolInfraRESTAPIKey   string
	StorageDSN           string
	OpenAIAPIKey         string
	OpenAIModel          string
	JitoBlockEngineURL   string
	JitoTipFloorURL      string
	JitoMinTipLamports   uint64
	KeypairPath          string
	WebhooksEnabled      bool
	WebhookSigningSecret string
	WebhookURL           string
}

func Load(envFile string) (*Config, error) {
	if envFile != "" {
		_ = godotenv.Load(envFile)
	}

	cfg := &Config{
		Env:                  getEnv("AEGIS_ENV", "development"),
		Cluster:              getEnv("AEGIS_CLUSTER", "mainnet-beta"),
		HTTPAddr:             getEnv("AEGIS_HTTP_ADDR", ":8080"),
		PolicyMode:           aegis.PolicyMode(strings.ToUpper(getEnv("AEGIS_POLICY_MODE", "SAFE"))),
		PublicAPIBaseURL:     getEnv("AEGIS_PUBLIC_API_BASE_URL", "http://localhost:8080"),
		SolanaRPCURL:         os.Getenv("SOLANA_RPC_URL"),
		SolanaWSURL:          os.Getenv("SOLANA_WS_URL"),
		SolanaRPCAPIKey:      os.Getenv("SOLANA_RPC_API_KEY"),
		YellowstoneGRPCURL:   os.Getenv("YELLOWSTONE_GRPC_URL"),
		YellowstoneGRPCToken: os.Getenv("YELLOWSTONE_GRPC_TOKEN"),
		SolInfraRESTAPIURL:   os.Getenv("SOLINFRA_REST_API_URL"),
		SolInfraRESTAPIKey:   os.Getenv("SOLINFRA_REST_API_KEY"),
		StorageDSN:           os.Getenv("DATABASE_URL"),
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:          getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		JitoBlockEngineURL:   getEnv("JITO_BLOCK_ENGINE_URL", "https://mainnet.block-engine.jito.wtf/api/v1"),
		JitoTipFloorURL:      getEnv("JITO_TIP_FLOOR_URL", "https://bundles.jito.wtf/api/v1/bundles/tip_floor"),
		JitoMinTipLamports:   getEnvUint64("JITO_MIN_TIP_LAMPORTS", 1000),
		KeypairPath:          os.Getenv("AEGIS_KEYPAIR_PATH"),
		WebhooksEnabled:      getEnvBool("AEGIS_WEBHOOKS_ENABLED", false),
		WebhookSigningSecret: os.Getenv("AEGIS_WEBHOOK_SIGNING_SECRET"),
		WebhookURL:           os.Getenv("AEGIS_WEBHOOK_URL"),
	}

	return cfg, Validate(cfg)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvUint64(key string, fallback uint64) uint64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func Validate(cfg *Config) error {
	var missing []string
	if cfg.SolanaRPCURL == "" {
		missing = append(missing, "SOLANA_RPC_URL")
	}
	if cfg.YellowstoneGRPCURL == "" {
		missing = append(missing, "YELLOWSTONE_GRPC_URL")
	}
	if cfg.YellowstoneGRPCToken == "" {
		missing = append(missing, "YELLOWSTONE_GRPC_TOKEN")
	}
	if cfg.StorageDSN == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.OpenAIAPIKey == "" {
		missing = append(missing, "OPENAI_API_KEY")
	}
	if cfg.KeypairPath == "" {
		missing = append(missing, "AEGIS_KEYPAIR_PATH")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	switch cfg.PolicyMode {
	case aegis.ModeFast, aegis.ModeSafe, aegis.ModeCheap, aegis.ModeAggressive:
	default:
		return fmt.Errorf("invalid AEGIS_POLICY_MODE: %s", cfg.PolicyMode)
	}
	return nil
}
