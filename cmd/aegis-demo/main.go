package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mira4sol/aegis/internal/config"
)

func main() {
	scenario := flag.String("scenario", "normal", "scenario: normal|expired|low-tip")
	flag.Parse()

	cfg, err := config.Load(".env")
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	base := cfg.PublicAPIBaseURL
	switch *scenario {
	case "normal":
		runCLI(base, []string{"-memo", "aegis-demo-normal"})
	case "expired":
		runCLI(base, []string{"-memo", "aegis-demo-expired", "-inject-expired"})
	case "low-tip":
		runCLI(base, []string{"-memo", "aegis-demo-low-tip", "-tip", "1"})
	default:
		fmt.Fprintf(os.Stderr, "unknown scenario %s\n", *scenario)
		os.Exit(1)
	}
	time.Sleep(2 * time.Second)
	fmt.Println("demo scenario dispatched")
}

func runCLI(base string, args []string) {
	// In production use exec.Command; kept simple for demo script parity.
	_ = context.Background()
	fmt.Printf("POST %s/v1/transactions scenario=%v\n", base, args)
}
