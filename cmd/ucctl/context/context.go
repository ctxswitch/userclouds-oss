package context

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"userclouds.com/cmd/ucctl/config"
)

// ListCommand lists all contexts
type ListCommand struct{}

func (c *ListCommand) RunE(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Contexts) == 0 {
		fmt.Println("No contexts configured")
		return nil
	}

	// Get sorted context names
	names := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	// Print contexts in a table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "CURRENT\tNAME\tURL\tCLIENT-ID")

	for _, name := range names {
		ctx := cfg.Contexts[name]
		current := " "
		if name == cfg.CurrentContext {
			current = "*"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", current, name, ctx.URL, ctx.ClientID)
	}

	return w.Flush()
}

// UseCommand switches to a different context
type UseCommand struct{}

func (c *UseCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ucctl context use <context-name>")
	}

	contextName := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.UseContext(contextName); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Switched to context %q\n", contextName)
	return nil
}

// SetCommand creates or updates a context
type SetCommand struct {
	URL          string
	ClientID     string
	ClientSecret string
}

func (c *SetCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ucctl context set <context-name> --url <url> --client-id <id> --client-secret <secret>")
	}

	contextName := args[0]

	if c.URL == "" {
		return fmt.Errorf("--url is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("--client-id is required")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("--client-secret is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := &config.Context{
		URL:          c.URL,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
	}

	cfg.SetContext(contextName, ctx)

	// If this is the first context, make it current
	if len(cfg.Contexts) == 1 {
		cfg.CurrentContext = contextName
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Context %q set\n", contextName)
	if cfg.CurrentContext == contextName {
		fmt.Printf("Switched to context %q\n", contextName)
	}

	return nil
}

// DeleteCommand deletes a context
type DeleteCommand struct{}

func (c *DeleteCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ucctl context delete <context-name>")
	}

	contextName := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.DeleteContext(contextName); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Context %q deleted\n", contextName)
	return nil
}

// ShowCommand displays the current context
type ShowCommand struct{}

func (c *ShowCommand) RunE(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.CurrentContext == "" {
		fmt.Println("No current context set")
		return nil
	}

	ctx, err := cfg.GetCurrentContext()
	if err != nil {
		return err
	}

	fmt.Printf("Current context: %s\n", cfg.CurrentContext)
	fmt.Printf("URL: %s\n", ctx.URL)
	fmt.Printf("Client ID: %s\n", ctx.ClientID)
	fmt.Printf("Client Secret: %s\n", maskSecret(ctx.ClientSecret))

	return nil
}

// maskSecret masks all but the last 4 characters of a secret
func maskSecret(secret string) string {
	if len(secret) <= 4 {
		return "****"
	}
	return "****" + secret[len(secret)-4:]
}