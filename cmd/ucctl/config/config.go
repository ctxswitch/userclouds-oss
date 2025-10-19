package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigDir is the default directory for ucctl config
	DefaultConfigDir = ".userclouds"
	// DefaultConfigFile is the default config file name
	DefaultConfigFile = "config.yaml"
	// LocalConfigFile is the local config file name
	LocalConfigFile = ".uc-context.yaml"
	// DefaultConfigPathVar is the default environment variable for config path
	DefaultConfigPathVar = "UC_CONTEXT"
)

// Config represents the entire ucctl configuration
type Config struct {
	// CurrentContext is the name of the currently active context
	CurrentContext string `yaml:"current_context,omitempty"`
	// Contexts is a map of context names to context configurations
	Contexts map[string]*Context `yaml:"contexts,omitempty"`
}

// Context represents a UserClouds environment configuration
type Context struct {
	// URL is the base URL of the UserClouds installation
	URL string `yaml:"url"`
	// ClientID is the OAuth2 client ID
	ClientID string `yaml:"client_id"`
	// ClientSecret is the OAuth2 client secret
	ClientSecret string `yaml:"client_secret"`
	// ConsoleTenant is the name of the console tenant context to use (for regular tenant contexts)
	ConsoleTenant string `yaml:"console,omitempty"`
	// IsConsoleTenant indicates if this context represents a console tenant
	IsConsoleTenant bool `yaml:"console_tenant,omitempty"`
}

// GetConfigPath returns the path to the config file with precedence order:
// 1. Explicit path parameter (if provided)
// 2. UC_CONTEXT environment variable
// 3. .uc-context.yaml in current directory
// 4. ~/.userclouds/config.yaml (default)
func GetConfigPath(explicitPath string) (string, error) {
	// Highest priority: explicit path from command-line flag
	if explicitPath != "" {
		return explicitPath, nil
	}

	// Second priority: UC_CONTEXT environment variable
	if envPath := os.Getenv(DefaultConfigPathVar); envPath != "" {
		return envPath, nil
	}

	// Third priority: .uc-context.yaml in current directory
	if _, err := os.Stat(LocalConfigFile); err == nil {
		return LocalConfigFile, nil
	}

	// Fourth priority (default): ~/.userclouds/config.yaml
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(home, DefaultConfigDir, DefaultConfigFile), nil
}

// Load loads the configuration using the config path precedence order
func Load(explicitPath string) (*Config, error) {
	path, err := GetConfigPath(explicitPath)
	if err != nil {
		return nil, err
	}

	return LoadFrom(path)
}

// LoadFrom loads the configuration from the specified path
func LoadFrom(path string) (*Config, error) {
	// If the file doesn't exist, return an empty config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{
			Contexts: make(map[string]*Context),
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize the contexts map if nil
	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]*Context)
	}

	return &cfg, nil
}

// Save saves the configuration using the config path precedence order
func (c *Config) Save(explicitPath string) error {
	path, err := GetConfigPath(explicitPath)
	if err != nil {
		return err
	}

	return c.SaveTo(path)
}

// SaveTo saves the configuration to the specified path
func (c *Config) SaveTo(path string) error {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetContext returns the context with the given name
func (c *Config) GetContext(name string) (*Context, error) {
	ctx, ok := c.Contexts[name]
	if !ok {
		return nil, fmt.Errorf("context %q not found", name)
	}
	return ctx, nil
}

// GetCurrentContext returns the currently active context
func (c *Config) GetCurrentContext() (*Context, error) {
	if c.CurrentContext == "" {
		return nil, fmt.Errorf("no current context set")
	}
	return c.GetContext(c.CurrentContext)
}

// SetContext adds or updates a context
func (c *Config) SetContext(name string, ctx *Context) {
	if c.Contexts == nil {
		c.Contexts = make(map[string]*Context)
	}
	c.Contexts[name] = ctx
}

// DeleteContext removes a context
func (c *Config) DeleteContext(name string) error {
	if _, ok := c.Contexts[name]; !ok {
		return fmt.Errorf("context %q not found", name)
	}

	// Only delete the context if it has no references
	tenants := make([]string, 0)
	for name, ctx := range c.Contexts {
		if ctx.ConsoleTenant == name {
			tenants = append(tenants, name)
		}
	}

	if len(tenants) != 0 {
		return fmt.Errorf("context %q still has referenced tenants [%q]", name, strings.Join(tenants, ","))
	}

	delete(c.Contexts, name)

	// If the deleted context was the current context, unset it
	if c.CurrentContext == name {
		c.CurrentContext = ""
	}

	return nil
}

// UseContext sets the current context
func (c *Config) UseContext(name string) error {
	if _, ok := c.Contexts[name]; !ok {
		return fmt.Errorf("context %q not found", name)
	}
	c.CurrentContext = name
	return nil
}

// GetConsoleTenantContext returns the console tenant context for the given context
// If the context is itself a console tenant, it returns the context itself
// Otherwise, it resolves the console tenant reference
func (c *Config) GetConsoleTenantContext(ctx *Context) (*Context, error) {
	if ctx.IsConsoleTenant {
		return ctx, nil
	}

	if ctx.ConsoleTenant == "" {
		return nil, fmt.Errorf("context does not have a console tenant reference")
	}

	consoleTenant, err := c.GetContext(ctx.ConsoleTenant)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve console tenant %q: %w", ctx.ConsoleTenant, err)
	}

	if !consoleTenant.IsConsoleTenant {
		return nil, fmt.Errorf("referenced context %q is not a console tenant", ctx.ConsoleTenant)
	}

	return consoleTenant, nil
}
