package cli

import (
	"context"
	"dossier/internal/config"
	"dossier/internal/core"
	"dossier/internal/harness"
	"dossier/internal/store"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	dossierHomeFlag string
	yesFlag         bool
)

// Execute runs the cobra command parser.
func Execute() {
	rootCmd := &cobra.Command{
		Use:   "dossier",
		Short: "Dossier: durable memory layer for agent-driven work",
	}

	rootCmd.PersistentFlags().StringVar(&dossierHomeFlag, "home", "", "Override default Dossier home directory")

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the Dossier workspace and config",
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error wiring service: %v\n", err)
				os.Exit(1)
			}

			res, err := svc.Init(context.Background(), yesFlag)
			if err != nil {
				fmt.Printf("Initialization failed: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Dossier initialized at %s\n\n", homeDir)

			dataMap, ok := res.Data.(map[string]any)
			if ok {
				tiers, ok := dataMap["harness_tiers"].(map[string]string)
				if ok {
					fmt.Println("Harness support:")
					names := []string{"claude-code", "codex", "antigravity"}
					harnessLabels := map[string]string{
						"claude-code": "Claude Code",
						"codex":       "Codex",
						"antigravity": "Antigravity",
					}
					descriptions := map[string]string{
						"claude-code": "Tier 1 candidate — MCP, hooks, transcript capture detected",
						"codex":       "Tier 2 candidate — MCP/hooks detected, transcript capture unavailable",
						"antigravity": "Tier 3 candidate — context/MCP fallback only",
					}
					for _, name := range names {
						tier := tiers[name]
						label := harnessLabels[name]
						desc := descriptions[name]
						fmt.Printf("- %s: %s (%s)\n", label, tier, desc)
					}
					fmt.Println()
				}
			}

			for _, warning := range res.Warnings {
				fmt.Printf("Warning: %s\n", warning)
			}
		},
	}
	initCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompts")

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Verify system health and configuration integrity",
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			res, err := svc.Doctor(context.Background())
			if err != nil {
				fmt.Printf("Doctor check failed: %v\n", err)
				os.Exit(1)
			}

			if res.OK {
				fmt.Println("Dossier workspace is healthy!")
			} else {
				fmt.Println("Dossier workspace checks failed.")
			}

			for _, warning := range res.Warnings {
				fmt.Printf("- Warning: %s\n", warning)
			}

			if !res.OK {
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(doctorCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func resolveHomeDir() string {
	if dossierHomeFlag != "" {
		return dossierHomeFlag
	}
	return config.Default().DossierHome
}

// Stub adapters for pure domains
type dummySearcher struct{}

func (d *dummySearcher) Search(ctx context.Context, query string, scope core.SearchScope) ([]core.Hit, error) {
	return nil, nil
}

type dummyTokenizer struct{}

func (d *dummyTokenizer) Estimate(text string) int {
	return 0
}

type realClock struct{}

func (r *realClock) Now() time.Time {
	return time.Now()
}

func wire(dossierHome string) (*core.Service, error) {
	cfgPath := filepath.Join(dossierHome, "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	// Write default config to disk if not exists
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := cfg.Save(cfgPath); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	}

	storeAdapter := store.NewFSStore(dossierHome)
	searchAdapter := &dummySearcher{}
	tokAdapter := &dummyTokenizer{}
	hregAdapter := harness.NewRegistry(dossierHome)
	clockAdapter := &realClock{}

	return core.NewService(storeAdapter, searchAdapter, tokAdapter, hregAdapter, clockAdapter, cfg.ToCoreConfig()), nil
}
