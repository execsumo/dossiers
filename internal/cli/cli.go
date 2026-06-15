package cli

import (
	"context"
	"dossier/internal/config"
	"dossier/internal/core"
	"dossier/internal/harness"
	"dossier/internal/mcp"
	"dossier/internal/search"
	"dossier/internal/store"
	"dossier/internal/tokenizer"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	dossierHomeFlag   string
	yesFlag           bool
	statusFlag        string
	jsonFlag          bool
	dossierSearchFlag string
	distilledFlag     string
	fromFileFlag      string
	forceFlag         bool
	sessionFlag       string
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

	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List open dossiers sorted by priority score",
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			res, err := svc.List(context.Background(), core.ListReq{Status: statusFlag})
			if err != nil {
				fmt.Printf("List failed: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				printJSON(res.Data)
				return
			}

			items, ok := res.Data.([]core.ListItem)
			if !ok {
				fmt.Printf("Unexpected data type returned: %T\n", res.Data)
				os.Exit(1)
			}

			if len(items) == 0 {
				fmt.Println("No dossiers found.")
				return
			}

			fmt.Printf("%-30s %-8s %-8s %-5s %s\n", "NAME/SLUG", "STATUS", "PRIORITY", "STALE", "NEXT ACTION")
			fmt.Println(strings.Repeat("-", 80))
			for _, item := range items {
				nameOrSlug := item.Name
				if nameOrSlug == "" {
					nameOrSlug = item.Slug
				}
				if len(nameOrSlug) > 28 {
					nameOrSlug = nameOrSlug[:25] + "..."
				}

				nextAction := item.NextAction
				if len(nextAction) > 28 {
					nextAction = nextAction[:25] + "..."
				}

				fmt.Printf("%-30s %-8s %-8d %-5d %s\n", nameOrSlug, item.Status, item.PriorityScore, item.StalenessDays, nextAction)
			}
		},
	}
	lsCmd.Flags().StringVar(&statusFlag, "status", "", "Filter by status (active|waiting|blocked|resolved|archived|all)")
	lsCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results in JSON format")

	showCmd := &cobra.Command{
		Use:   "show <slug-or-id>",
		Short: "Show a dossier's details and distilled state",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			res, err := svc.Recall(context.Background(), core.RecallReq{ID: args[0]})
			if err != nil {
				fmt.Printf("Error showing dossier: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				printJSON(res.Data)
				return
			}

			recall, ok := res.Data.(core.RecallResult)
			if !ok {
				fmt.Printf("Unexpected data type returned: %T\n", res.Data)
				os.Exit(1)
			}

			fmt.Printf("Name:           %s\n", recall.Frontmatter.Name)
			fmt.Printf("ID:             %s\n", recall.Frontmatter.ID)
			fmt.Printf("Slug:           %s\n", recall.Frontmatter.Slug)
			fmt.Printf("Status:         %s\n", recall.Frontmatter.Status)
			fmt.Printf("Importance:     %s\n", recall.Frontmatter.Importance)
			fmt.Printf("Urgency:        %s\n", recall.Frontmatter.Urgency)
			if recall.Frontmatter.DueDate != "" {
				fmt.Printf("Due Date:       %s\n", recall.Frontmatter.DueDate)
			}
			fmt.Printf("Token Estimate: %d\n", recall.TokenEstimate)
			fmt.Printf("Next Action:    %s\n", recall.Frontmatter.NextAction)
			if len(recall.Frontmatter.OpenQuestions) > 0 {
				fmt.Println("Open Questions:")
				for _, q := range recall.Frontmatter.OpenQuestions {
					fmt.Printf("  - %s\n", q)
				}
			}
			fmt.Println(strings.Repeat("-", 80))
			fmt.Println(recall.DistilledState)
		},
	}
	showCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results in JSON format")

	pathCmd := &cobra.Command{
		Use:   "path [<slug-or-id>]",
		Short: "Get the directory path of a dossier or the workspace root",
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			if len(args) == 0 {
				if jsonFlag {
					printJSON(map[string]string{"path": homeDir})
				} else {
					fmt.Println(homeDir)
				}
				return
			}

			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			res, err := svc.Path(context.Background(), core.PathReq{ID: args[0]})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				printJSON(map[string]string{"path": res.Data.(string)})
			} else {
				fmt.Println(res.Data)
			}
		},
	}
	pathCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results in JSON format")

	archiveCmd := &cobra.Command{
		Use:   "archive <slug-or-id>",
		Short: "Archive a dossier (marks status as archived, keeping files)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			res, err := svc.Archive(context.Background(), core.ArchiveReq{ID: args[0]})
			if err != nil {
				fmt.Printf("Archive failed: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				printJSON(res)
				return
			}

			fmt.Printf("Dossier archived successfully. New revision: %s\n", res.Data.(core.Revision))
		},
	}
	archiveCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results in JSON format")

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search distilled state and artifacts across dossiers",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			req := core.SearchReq{
				Query: args[0],
				Scope: core.SearchScope{
					DossierID: dossierSearchFlag,
				},
			}

			res, err := svc.Search(context.Background(), req)
			if err != nil {
				fmt.Printf("Search failed: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				printJSON(res.Data)
				return
			}

			hits, ok := res.Data.([]core.Hit)
			if !ok {
				fmt.Printf("Unexpected data type returned: %T\n", res.Data)
				os.Exit(1)
			}

			if len(hits) == 0 {
				fmt.Println("No matches found.")
				return
			}

			for i, hit := range hits {
				fmt.Printf("Dossier:  %s (%s)\n", hit.DossierName, hit.DossierID)
				if hit.ArtifactID != "" {
					fmt.Printf("Artifact: %s (%s)\n", hit.Title, hit.ArtifactID)
				}
				fmt.Printf("File:     %s\n", hit.Path)
				if hit.LineNumber > 0 {
					fmt.Printf("Line %d:  %s\n", hit.LineNumber, hit.Snippet)
				} else {
					fmt.Printf("Match:    %s\n", hit.Snippet)
				}
				if i < len(hits)-1 {
					fmt.Println(strings.Repeat("-", 80))
				}
			}
		},
	}
	searchCmd.Flags().StringVarP(&dossierSearchFlag, "dossier", "d", "", "Scope search to a specific dossier (slug or ID)")
	searchCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results in JSON format")

	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Manage the generated open-work context",
	}

	contextRefreshCmd := &cobra.Command{
		Use:   "refresh",
		Short: "Regenerate the context library",
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			res, err := svc.ContextRefresh(context.Background())
			if err != nil {
				fmt.Printf("Context refresh failed: %v\n", err)
				os.Exit(1)
			}

			if !res.OK {
				fmt.Println("Context refresh failed.")
				os.Exit(1)
			}

			fmt.Println("Context library regenerated successfully.")
		},
	}
	contextCmd.AddCommand(contextRefreshCmd)

	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage the MCP server interface",
	}

	mcpServeCmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server over stdio",
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			server := mcp.NewServer(svc, os.Stdin, os.Stdout)
			if err := server.Run(context.Background()); err != nil {
				fmt.Printf("MCP server exited with error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	mcpCmd.AddCommand(mcpServeCmd)

	promoteCmd := &cobra.Command{
		Use:   "promote <name>",
		Short: "Promote a new dossier from session content or file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			var content string
			if fromFileFlag != "" {
				data, err := os.ReadFile(fromFileFlag)
				if err != nil {
					fmt.Printf("Error reading file: %v\n", err)
					os.Exit(1)
				}
				content = string(data)
			}

			req := core.PromoteReq{
				Name:                   args[0],
				DistilledStateMarkdown: distilledFlag,
				Content:                content,
				Force:                  forceFlag || yesFlag,
			}

			res, err := svc.Promote(context.Background(), req)
			if err != nil {
				if dErr, ok := err.(*core.DomainError); ok && dErr.Code == core.ErrAmbiguousTarget {
					if jsonFlag {
						printJSON(res)
						return
					}
					fmt.Println("Error: Multiple likely dossiers match this name. Disambiguation required:")
					suggestions := res.Data.([]core.Suggestion)
					for _, sug := range suggestions {
						fmt.Printf("- %s (ID: %s, Confidence: %s) - Reason: %s\n", sug.Name, sug.ID, sug.Confidence, sug.Reason)
					}
					fmt.Println("\nTo create anyway, re-run with --force or -y.")
					os.Exit(1)
				}

				fmt.Printf("Promote failed: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				printJSON(res)
				return
			}

			fmt.Printf("Dossier promoted successfully. ID: %s\n", res.Data.(string))
		},
	}
	promoteCmd.Flags().StringVar(&distilledFlag, "distilled", "", "Distilled state markdown body")
	promoteCmd.Flags().StringVar(&fromFileFlag, "from-file", "", "Path to session content file")
	promoteCmd.Flags().BoolVar(&forceFlag, "force", false, "Force create dossier even if matches exist")
	promoteCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results in JSON format")

	linkCmd := &cobra.Command{
		Use:   "link [<slug-or-id>]",
		Short: "Link session content or file to a dossier",
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			var content string
			var title string
			if fromFileFlag != "" {
				data, err := os.ReadFile(fromFileFlag)
				if err != nil {
					fmt.Printf("Error reading file: %v\n", err)
					os.Exit(1)
				}
				content = string(data)
				title = filepath.Base(fromFileFlag)
			}

			var targetID string
			if len(args) > 0 {
				targetID = args[0]
			}

			req := core.LinkReq{
				ID:      targetID,
				Content: content,
				Title:   title,
			}

			res, err := svc.Link(context.Background(), req)
			if err != nil {
				if dErr, ok := err.(*core.DomainError); ok && dErr.Code == core.ErrAmbiguousTarget {
					if jsonFlag {
						printJSON(res.Data)
						return
					}
					fmt.Println("Ambiguity detected. Top matching dossiers for this content:")
					suggestions := res.Data.([]core.Suggestion)
					for _, sug := range suggestions {
						fmt.Printf("- %s (ID: %s, Confidence: %s) - Reason: %s\n", sug.Name, sug.ID, sug.Confidence, sug.Reason)
					}
					fmt.Println("\nTo link, run again with: dossier link <id> --from-file <path>")
					os.Exit(1)
				}

				fmt.Printf("Link failed: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				printJSON(res)
				return
			}

			fmt.Printf("Dossier linked successfully. New revision: %s\n", res.Data.(core.Revision))
		},
	}
	linkCmd.Flags().StringVar(&fromFileFlag, "from-file", "", "Path to session content file")
	linkCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results in JSON format")

	activeCmd := &cobra.Command{
		Use:   "active",
		Short: "Show the active dossier bound to the current session",
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			sessID := resolveSessionID()
			res, err := svc.Active(context.Background(), core.ActiveReq{SessionID: sessID})
			if err != nil {
				if jsonFlag {
					printJSON(map[string]any{"ok": false, "error": err.Error()})
					os.Exit(1)
				}
				fmt.Printf("No active dossier bound to this session: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				printJSON(res.Data)
				return
			}

			binding := res.Data.(*core.SessionBinding)
			fmt.Printf("Active Dossier ID:  %s\n", binding.DossierID)
			fmt.Printf("Bound At:           %s\n", binding.BoundAt.Format(time.RFC3339))
			fmt.Printf("Last Seen Revision: %s\n", binding.LastSeenRevision)
		},
	}
	activeCmd.Flags().StringVar(&sessionFlag, "session", "", "Session ID to check")
	activeCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results in JSON format")

	switchCmd := &cobra.Command{
		Use:   "switch <slug-or-id>",
		Short: "Switch the active dossier binding for the session",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			sessID := resolveSessionID()
			res, err := svc.Switch(context.Background(), core.SwitchReq{ID: args[0], SessionID: sessID})
			if err != nil {
				fmt.Printf("Switch failed: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				printJSON(res.Data)
				return
			}

			recall := res.Data.(core.RecallResult)
			fmt.Printf("Switched active dossier to: %s (%s)\n", recall.Frontmatter.Name, recall.Frontmatter.ID)
			fmt.Printf("Revision: %s\n", recall.Revision)
		},
	}
	switchCmd.Flags().StringVar(&sessionFlag, "session", "", "Session ID to bind")
	switchCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results in JSON format")

	hookCmd := &cobra.Command{
		Use:   "hook <session-start|session-end|pre-compaction>",
		Short: "Run lifecycle integration hooks",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			homeDir := resolveHomeDir()
			svc, err := wire(homeDir)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			var payload struct {
				SessionID      string `json:"session_id"`
				HookEventName  string `json:"hook_event_name"`
				Transcript     string `json:"transcript"`
				DistilledState string `json:"distilled_state"`
			}

			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				dec := json.NewDecoder(os.Stdin)
				_ = dec.Decode(&payload)
			}

			sessID := payload.SessionID
			if sessID == "" {
				sessID = resolveSessionID()
			}

			switch args[0] {
			case "session-start":
				resText, err := svc.SessionStart(context.Background(), sessID)
				if err != nil {
					fmt.Printf("Session start hook failed: %v\n", err)
					os.Exit(1)
				}
				fmt.Print(resText)

			case "session-end", "pre-compaction":
				err := svc.SessionEnd(context.Background(), sessID, payload.DistilledState, payload.Transcript)
				if err != nil {
					fmt.Printf("Session end hook failed: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Session hook completed successfully.")

			default:
				fmt.Printf("Unknown hook event: %s\n", args[0])
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(pathCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(promoteCmd)
	rootCmd.AddCommand(linkCmd)
	rootCmd.AddCommand(activeCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(hookCmd)

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

func resolveSessionID() string {
	if sessionFlag != "" {
		return sessionFlag
	}
	if envSess := os.Getenv("DOSSIER_SESSION"); envSess != "" {
		return envSess
	}
	return "sess_default"
}

func printJSON(data any) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonBytes))
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

	var searchAdapter core.Searcher
	if search.IsRipgrepAvailable() {
		searchAdapter = search.NewRipgrepSearcher(dossierHome)
	} else {
		searchAdapter = search.NewNativeSearcher(dossierHome)
	}

	tokAdapter, err := tokenizer.NewBPETokenizer()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize BPE tokenizer: %w", err)
	}

	hregAdapter := harness.NewRegistry(dossierHome)
	clockAdapter := &realClock{}

	return core.NewService(storeAdapter, searchAdapter, tokAdapter, hregAdapter, clockAdapter, cfg.ToCoreConfig()), nil
}
