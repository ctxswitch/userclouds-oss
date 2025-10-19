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
	cfg, err := config.Load("")
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
	fmt.Fprintln(w, "CURRENT\tNAME\tTYPE\tURL")

	for _, name := range names {
		ctx := cfg.Contexts[name]
		current := " "
		if name == cfg.CurrentContext {
			current = "*"
		}
		contextType := "tenant"
		if ctx.IsConsoleTenant {
			contextType = "console"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", current, name, contextType, ctx.URL)
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

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.UseContext(contextName); err != nil {
		return err
	}

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Switched to context %q\n", contextName)
	return nil
}

// SetCommand creates or updates a context
type SetCommand struct {
	URL      string
	ClientID string
	// TODO: this should be a secret string.
	ClientSecret    string
	Console         string
	IsConsoleTenant bool
}

func (c *SetCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ucctl context set <context-name> --url <url> --client-id <id> --client-secret <secret> [--console <console-tenant-name> | --console-tenant]")
	}

	contextName := args[0]

	// Validate required fields
	if c.URL == "" {
		return fmt.Errorf("--url is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("--client-id is required")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("--client-secret is required")
	}

	// Validate console tenant logic
	if c.IsConsoleTenant && c.Console != "" {
		return fmt.Errorf("cannot specify both --console-tenant and --console (console tenants cannot reference other consoles)")
	}

	if !c.IsConsoleTenant && c.Console == "" {
		return fmt.Errorf("--console is required for regular tenant contexts (or use --console-tenant to create a console tenant)")
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// If this is a regular tenant context, validate that the console reference exists
	if !c.IsConsoleTenant {
		consoleCtx, err := cfg.GetContext(c.Console)
		if err != nil {
			return fmt.Errorf("console tenant %q not found. Create it first with --console-tenant flag", c.Console)
		}
		if !consoleCtx.IsConsoleTenant {
			return fmt.Errorf("referenced context %q is not a console tenant", c.Console)
		}
	}

	ctx := &config.Context{
		URL:             c.URL,
		ClientID:        c.ClientID,
		ClientSecret:    c.ClientSecret,
		ConsoleTenant:   c.Console,
		IsConsoleTenant: c.IsConsoleTenant,
	}

	cfg.SetContext(contextName, ctx)

	// If this is the first context, make it current
	if len(cfg.Contexts) == 1 {
		cfg.CurrentContext = contextName
	}

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	contextType := "tenant"
	if c.IsConsoleTenant {
		contextType = "console tenant"
	}
	fmt.Printf("%s context %q set\n", contextType, contextName)
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

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.DeleteContext(contextName); err != nil {
		return err
	}

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Context %q deleted\n", contextName)
	return nil
}

// ShowCommand displays the current context
type ShowCommand struct{}

func (c *ShowCommand) RunE(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
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

	contextType := "Tenant"
	if ctx.IsConsoleTenant {
		contextType = "Console Tenant"
	}

	fmt.Printf("Current context: %s\n", cfg.CurrentContext)
	fmt.Printf("Type: %s\n", contextType)
	fmt.Printf("URL: %s\n", ctx.URL)
	fmt.Printf("Client ID: %s\n", ctx.ClientID)
	fmt.Printf("Client Secret: %s\n", maskSecret(ctx.ClientSecret))

	if !ctx.IsConsoleTenant && ctx.ConsoleTenant != "" {
		fmt.Printf("Console Tenant: %s\n", ctx.ConsoleTenant)

		// Resolve and show console tenant details
		consoleCtx, err := cfg.GetConsoleTenantContext(ctx)
		if err != nil {
			fmt.Printf("  (Warning: %v)\n", err)
		} else {
			fmt.Printf("  Console URL: %s\n", consoleCtx.URL)
			fmt.Printf("  Console Client ID: %s\n", consoleCtx.ClientID)
		}
	}

	return nil
}

// maskSecret masks all but the last 4 characters of a secret
func maskSecret(secret string) string {
	if len(secret) <= 4 {
		return "****"
	}
	return "****" + secret[len(secret)-4:]
}
