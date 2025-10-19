package common

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"userclouds.com/cmd/ucctl/config"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/secret"
)

const (
	// DefaultClientSecretVar is the default environment variable for client secrets
	DefaultClientSecretVar = "UC_CLIENT_SECRET"
	// DefaultBearerTokenVar is the default environment variable for bearer tokens
	DefaultBearerTokenVar = "UC_BEARER_TOKEN"
)

// AuthType represents the type of authentication to use
type AuthType string

const (
	// AuthTypeClientCredentials uses OAuth2 client credentials flow
	AuthTypeClientCredentials AuthType = "client-credentials"
	// AuthTypeBearerToken uses a bearer token for authentication
	AuthTypeBearerToken AuthType = "bearer-token"
	// AuthTypeTokenSource uses a custom OIDC token source
	AuthTypeTokenSource AuthType = "token-source"
)

// Credentials holds the authentication credentials for a UserClouds tenant
type Credentials struct {
	URL          string
	ClientID     string
	ClientSecret string
	BearerToken  string
	TokenSource  oidc.TokenSource
	AuthType     AuthType
}

// LoadCredentialsFromContext loads credentials from the current context or explicit flags
// If no explicit credentials are provided, it automatically uses the current context
func LoadCredentialsFromContext(url, clientID, clientSecret, clientSecretVar string) (*Credentials, error) {
	// If no explicit credentials provided, load from config
	if url == "" && clientID == "" && clientSecret == "" {
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}

		ctx, err := cfg.GetCurrentContext()
		if err != nil {
			return nil, fmt.Errorf("no current context set. Use 'ucctl context use <name>' or provide --url, --client-id, and --client-secret")
		}

		// Use current context directly
		if url == "" {
			url = ctx.URL
		}
		if clientID == "" {
			clientID = ctx.ClientID
		}
		if clientSecret == "" {
			clientSecret = ctx.ClientSecret
		}
	}

	if url == "" {
		return nil, fmt.Errorf("URL is required (use --url or set a context)")
	}

	if clientID == "" {
		return nil, fmt.Errorf("client ID is required (use --client-id or set a context)")
	}

	// Get client secret from environment variable if not set directly
	if clientSecret == "" {
		if clientSecretVar == "" {
			clientSecretVar = DefaultClientSecretVar
		}
		clientSecret = os.Getenv(clientSecretVar)
		if clientSecret == "" {
			return nil, fmt.Errorf("client secret is required (use --client-secret, set %s env var, or use a context)", clientSecretVar)
		}
	}

	return &Credentials{
		URL:          url,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthType:     AuthTypeClientCredentials,
	}, nil
}

// LoadCredentialsFromContextName loads credentials from a named context
// This is useful when you need to load credentials for a specific context (not the current one)
func LoadCredentialsFromContextName(contextName string) (*Credentials, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	ctx, err := cfg.GetContext(contextName)
	if err != nil {
		return nil, fmt.Errorf("context %q not found: %w", contextName, err)
	}

	return &Credentials{
		URL:          ctx.URL,
		ClientID:     ctx.ClientID,
		ClientSecret: ctx.ClientSecret,
		AuthType:     AuthTypeClientCredentials,
	}, nil
}

// LoadCredentialsFromEnv loads credentials from explicit values with client secret from environment variable
// This is useful for the sync command which uses environment variables for secrets
func LoadCredentialsFromEnv(url, clientID, clientSecretVar string) (*Credentials, error) {
	if url == "" {
		return nil, fmt.Errorf("URL is required")
	}

	if clientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	if clientSecretVar == "" {
		clientSecretVar = DefaultClientSecretVar
	}

	clientSecret := os.Getenv(clientSecretVar)
	if clientSecret == "" {
		return nil, fmt.Errorf("client secret not found in environment variable %s", clientSecretVar)
	}

	return &Credentials{
		URL:          url,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthType:     AuthTypeClientCredentials,
	}, nil
}

// LoadCredentialsWithBearerToken creates credentials using a bearer token
func LoadCredentialsWithBearerToken(url, bearerToken string) (*Credentials, error) {
	if url == "" {
		return nil, fmt.Errorf("URL is required")
	}

	if bearerToken == "" {
		return nil, fmt.Errorf("bearer token is required")
	}

	return &Credentials{
		URL:         url,
		BearerToken: bearerToken,
		AuthType:    AuthTypeBearerToken,
	}, nil
}

// LoadCredentialsWithBearerTokenFromEnv loads credentials using a bearer token from environment variable
func LoadCredentialsWithBearerTokenFromEnv(url, bearerTokenVar string) (*Credentials, error) {
	if url == "" {
		return nil, fmt.Errorf("URL is required")
	}

	if bearerTokenVar == "" {
		bearerTokenVar = DefaultBearerTokenVar
	}

	bearerToken := os.Getenv(bearerTokenVar)
	if bearerToken == "" {
		return nil, fmt.Errorf("bearer token not found in environment variable %s", bearerTokenVar)
	}

	return &Credentials{
		URL:         url,
		BearerToken: bearerToken,
		AuthType:    AuthTypeBearerToken,
	}, nil
}

// LoadCredentialsWithTokenSource creates credentials using a custom OIDC token source
func LoadCredentialsWithTokenSource(url string, tokenSource oidc.TokenSource) (*Credentials, error) {
	if url == "" {
		return nil, fmt.Errorf("URL is required")
	}

	if tokenSource == nil {
		return nil, fmt.Errorf("token source is required")
	}

	return &Credentials{
		URL:         url,
		TokenSource: tokenSource,
		AuthType:    AuthTypeTokenSource,
	}, nil
}

// GetClientCredentials creates a jsonclient.Option for authentication
// This method supports multiple authentication types:
// - OAuth2 client credentials flow
// - Bearer token authentication
// - Custom OIDC token source
func (c *Credentials) GetClientCredentials() (jsonclient.Option, error) {
	switch c.AuthType {
	case AuthTypeClientCredentials:
		if c.ClientID == "" || c.ClientSecret == "" {
			return nil, fmt.Errorf("client ID and secret are required for client credentials authentication")
		}
		return jsonclient.ClientCredentialsForURL(c.URL, c.ClientID, c.ClientSecret, nil)

	case AuthTypeBearerToken:
		if c.BearerToken == "" {
			return nil, fmt.Errorf("bearer token is required for bearer token authentication")
		}
		return jsonclient.HeaderAuthBearer(c.BearerToken), nil

	case AuthTypeTokenSource:
		if c.TokenSource == nil {
			return nil, fmt.Errorf("token source is required for token source authentication")
		}
		return jsonclient.TokenSource(c.TokenSource), nil

	default:
		return nil, fmt.Errorf("unknown authentication type: %s", c.AuthType)
	}
}

// OIDCLoginOptions holds options for interactive OIDC login
type OIDCLoginOptions struct {
	TenantURL     string   // UserClouds tenant URL
	ClientID      string   // OAuth2 client ID
	ClientSecret  string   // OAuth2 client secret (optional for public clients)
	RedirectURL   string   // Redirect URL (defaults to http://localhost:8080/callback)
	Scopes        []string // OAuth2 scopes (defaults to openid, profile, email)
	CallbackPort  int      // Port for local callback server (defaults to 8080)
	BrowserOpener func(url string) error // Custom browser opener (optional)
}

// LoadCredentialsWithOIDCLogin performs an interactive OIDC login flow
// This will:
// 1. Start a local HTTP server to handle the OAuth callback
// 2. Open the user's browser to the OIDC provider's authorization page
// 3. Wait for the callback with the authorization code
// 4. Exchange the code for tokens
// 5. Return credentials with the access token
func LoadCredentialsWithOIDCLogin(ctx context.Context, opts OIDCLoginOptions) (*Credentials, error) {
	// Set defaults
	if opts.RedirectURL == "" {
		if opts.CallbackPort == 0 {
			opts.CallbackPort = 8080
		}
		opts.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", opts.CallbackPort)
	}
	if len(opts.Scopes) == 0 {
		opts.Scopes = []string{"openid", "profile", "email"}
	}
	if opts.BrowserOpener == nil {
		opts.BrowserOpener = openBrowser
	}

	// Create OIDC authenticator
	clientSecretStr := secret.NewTestString(opts.ClientSecret)
	authn, err := oidc.NewAuthenticator(
		ctx,
		opts.TenantURL,
		opts.ClientID,
		clientSecretStr,
		opts.RedirectURL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC authenticator: %w", err)
	}

	// Override scopes if provided
	if len(opts.Scopes) > 0 {
		authn.Config.Scopes = opts.Scopes
	}

	// Generate state and nonce for CSRF protection
	state := fmt.Sprintf("state-%d", time.Now().Unix())
	nonce := fmt.Sprintf("nonce-%d", time.Now().Unix())

	// Start local callback server
	callbackCh := make(chan *oidc.TokenInfo, 1)
	errorCh := make(chan error, 1)

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", opts.CallbackPort),
	}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		tokenInfo, statusCode, err := authn.ProcessAuthCodeCallback(r, state)
		if err != nil {
			errorCh <- fmt.Errorf("failed to process callback (status %d): %w", statusCode, err)
			http.Error(w, "Authentication failed", http.StatusInternalServerError)
			return
		}

		callbackCh <- tokenInfo

		// Show success page
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<html>
			<head><title>Authentication Successful</title></head>
			<body style="font-family: sans-serif; text-align: center; padding-top: 50px;">
				<h1>✓ Authentication Successful</h1>
				<p>You can close this window and return to the terminal.</p>
			</body>
			</html>
		`)
	})

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorCh <- fmt.Errorf("failed to start callback server: %w", err)
		}
	}()

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Build authorization URL
	authURL := authn.Config.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce))

	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If the browser doesn't open automatically, visit:\n%s\n\n", authURL)

	// Open browser
	if err := opts.BrowserOpener(authURL); err != nil {
		fmt.Printf("Warning: failed to open browser automatically: %v\n", err)
	}

	// Wait for callback or timeout
	var tokenInfo *oidc.TokenInfo
	select {
	case tokenInfo = <-callbackCh:
		// Success!
	case err := <-errorCh:
		server.Shutdown(ctx)
		return nil, err
	case <-time.After(5 * time.Minute):
		server.Shutdown(ctx)
		return nil, fmt.Errorf("authentication timeout after 5 minutes")
	case <-ctx.Done():
		server.Shutdown(ctx)
		return nil, fmt.Errorf("authentication cancelled")
	}

	// Shutdown server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	fmt.Printf("Authentication successful!\n")

	// Return credentials with the access token
	return &Credentials{
		URL:         opts.TenantURL,
		BearerToken: tokenInfo.AccessToken,
		AuthType:    AuthTypeBearerToken,
	}, nil
}

// openBrowser opens the specified URL in the user's default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// AddAuthFlags adds standard authentication flags to a command
// Note: Commands automatically use the current context if no explicit credentials are provided
func AddAuthFlags(cmd *cobra.Command, url, clientID, clientSecret, clientSecretVar *string, authnType *string) {
	cmd.Flags().StringVarP(url, "url", "", "", "Tenant URL (overrides context)")
	cmd.Flags().StringVarP(clientID, "client-id", "", "", "client ID (overrides context)")
	cmd.Flags().StringVarP(clientSecret, "client-secret", "", "", "client secret (overrides context)")
	cmd.Flags().StringVarP(clientSecretVar, "client-secret-var", "", DefaultClientSecretVar, "environment variable containing client secret")
	cmd.Flags().StringVarP(authnType, "authn-type", "", "", "authentication type filter (social, password)")
}