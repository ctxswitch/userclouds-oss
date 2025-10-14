package secret

import (
	"fmt"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret/prefix"
	"userclouds.com/infra/secret/provider"
)

// getSecretPath returns a constructed path based on several attributes.
func getSecretPath(uv universe.Universe, serviceName, name string) string {
	// This used to differentiate paths between on-prem and the hosted cloud.  For OSS this
	// isn't an issue anymore, so we just pass through the 'userclouds' prefixed path.
	// TODO: We should look at getting rid of uv in the path.
	// TODO: Question, should the uc prefix only be done when we are using external SM?  Currently
	//   this is only used by LocationFromName which is only called from m2m/auth, so might be ok.
	return fmt.Sprintf("userclouds/%s/%s/%s", uv, serviceName, name)
}

// LocationFromName returns a full secret name/location with the correct universe formatting
// Prefixed with `userclouds` for our on-prem usage to allow us to namespace in customer SM.
func LocationFromName(serviceName, name string) string {
	// This may break some previous assumptions, but since on-prem and cloud universes are
	// aws by default, this shouldn't break.  Local development work will need to define
	// the specific manager instead of intuiting it from the universe.
	// TODO: Update auth/m2m to respect errors.
	pv, _ := provider.FromEnv()
	path := getSecretPath(universe.Current(), serviceName, name)
	return fmt.Sprintf("%s%s", pv.Prefix(), path)
}

// FromLocation returns a new secret.String with the specified location
func FromLocation(location string) *String {
	return &String{location: location}
}

// NewTestString returns a string that is *not* stored in AWS Secret Manager
func NewTestString(s string) String {
	return String{location: fmt.Sprintf("%s%s", prefix.PrefixDevLiteral, s)}
}
