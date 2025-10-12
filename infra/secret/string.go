package secret

import (
	"context"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"strings"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret/prefix"
	"userclouds.com/infra/secret/provider"
	"userclouds.com/infra/ucerr"
)

var (
	// UIPlaceholder is a placeholder for UIs to use when displaying secrets
	UIPlaceholder = String{location: "********"}
	// EmptyString is a secret.String that is empty (for UIs mostly)
	EmptyString = String{location: ""}
)

// String is a string value that is potentially secured by an underlying provider
type String struct {
	location string // the location, which may be the secret or a prefixed pointer
	provider provider.Interface
}

// NewString returns a new secret.String that is stored "correctly" according to
// the underlying provider.
func NewString(ctx context.Context, serviceName, name, secret string) (*String, error) {
	// Special case that existed prior to refactor where empty secrets could be added. It
	// may makes sense to phase these out.
	if secret == "" {
		return &EmptyString, nil
	}

	uv := universe.Current()

	pv := provider.FromEnv()
	path := getSecretPath(uv, serviceName, name)

	err := pv.Save(ctx, path, secret)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// There's a special condition for dev-like providers that do not store the path.  I think
	// that we may be able to get rid of the "dev" provider with the new local development
	// patterns, but I'll keep this here to be compatible for now.  Dev literals are not supported
	// for NewString, but this mirrors the capabilities of the original.
	var loc string
	if pv.IsDev() {
		encSecret := base64.StdEncoding.EncodeToString([]byte(secret))
		loc = fmt.Sprintf("%s%s", pv.Prefix(), encSecret)
	} else {
		loc = fmt.Sprintf("%s%s", pv.Prefix(), path)
	}

	return FromLocation(loc), nil
}

// Resolve decides if the string is a Secret Store path and resolves it, or returns
// the string unchanged otherwise.
func (s *String) Resolve(ctx context.Context) (string, error) {
	if s.IsEmpty() {
		return "", nil
	}

	secret, found := c.Get(s.location)
	if found {
		return secret, nil
	}

	pv, err := s.GetProvider()
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	px, err := prefix.PrefixFromString(pv.Prefix())
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	value, err := pv.Get(ctx, px.Value(s.location))
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	// TODO: I've removed the handling of secrets that did not have a prefix.  Add them back later.
	//   This will also handle empty secrets as well.

	c.Store(s.location, value)
	return value, nil
}

// ResolveForUI simplifies the logic (slightly) for UIs to display secrets
func (s *String) ResolveForUI(ctx context.Context) (*String, error) {
	secret, err := s.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if secret == "" {
		return &EmptyString, nil
	}

	return &UIPlaceholder, nil
}

// ResolveInsecurelyForUI is currently used for login app client secrets only,
// when the user actually does need to see them in the UI (rather than just update them)
func (s *String) ResolveInsecurelyForUI(ctx context.Context) (*String, error) {
	secret, err := s.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &String{location: secret}, nil
}

// Delete removes the secret from the secret store (if applicable) and clears the location
func (s *String) Delete(ctx context.Context) error {
	var err error

	pv, err := s.GetProvider()
	if err != nil {
		return ucerr.Wrap(err)
	}

	px, err := prefix.PrefixFromString(pv.Prefix())
	if err != nil {
		return ucerr.Wrap(err)
	}

	err = pv.Delete(ctx, px.Value(s.location))
	if err != nil {
		return ucerr.Wrap(err)
	}

	s.location = ""
	return ucerr.Wrap(err)
}

// UnmarshalYAML implements yaml.Unmarshaler
// Note like UnmarshalText we assume this is a location, and
// we'll lazily resolve it later as needed
func (s *String) UnmarshalYAML(unmarshal func(any) error) error {
	// the secret path itself is just a string, so start there
	var uri string
	if err := unmarshal(&uri); err != nil {
		return ucerr.Wrap(err)
	}
	s.location = uri
	return nil
}

// MarshalText implements encoding.TextMarshaler
// NB: we don't implement MarshalJSON because we intentionally *don't* want
// to emit a rich object here (for backcompat, and no need)
func (s *String) MarshalText() ([]byte, error) {
	// we always save location since it's either the pointer we
	// want to save, or it's a copy of .value anyway
	return []byte(s.location), nil
}

// UnmarshalText implements json.Unmarshaler
// We always assume this is a location, and we'll lazily resolve
// the secret later in Resolve() as needed
func (s *String) UnmarshalText(b []byte) error {
	s.location = string(b)
	return nil
}

// String implements Stringer, specifically to obscure secrets when logged
// To actually use a secret, you need to explicitly use Resolve()
func (s *String) String() string {
	return strings.Repeat("*", len(s.location))
}

// Validate implements Validateable
func (s *String) Validate() error {
	// empty secrets are ok
	if s.IsEmpty() {
		return nil
	}

	px, err := prefix.PrefixFromString(s.location)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := px.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	// passthrough secrets are ok again for this migration
	return nil
}

// IsEmpty checks if the secret.String location is empty
func (s *String) IsEmpty() bool {
	return s.location == ""

}

// Scan implements sql.Scanner
func (s *String) Scan(value any) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		s.location = v
		return nil
	default:
		return ucerr.Errorf("cannot scan %T into secret.String", value)
	}
}

// Value implements sql.Valuer
func (s *String) Value() (driver.Value, error) {
	return s.location, nil
}

// GetProvider returns the secret manager provider associated with the secret.
func (s *String) GetProvider() (provider.Interface, error) {
	if s.provider != nil {
		return s.provider, nil
	}

	pv, err := provider.FromLocation(s.location)
	if err != nil {
		return nil, err
	}

	s.provider = pv
	return s.provider, nil
}

// WithProvider sets the provider that will be used for storing the secret.  Currently
// the provider is intuited from the string location, but this allows us to override
// it for location based discoveries.
func (s *String) WithProvider(provider provider.Interface) *String {
	// TODO: I'm not entirely fond of this approach, but it works well for testing (which
	//	 is only where this is used at the moment).  This would be nicer if we relied on this
	//	 a little more when the string was initialized.  Integrate this into NewString and
	//   the other String creation functions.
	s.provider = provider
	return s
}
