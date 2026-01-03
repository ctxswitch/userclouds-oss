package env

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvider_Get(t *testing.T) {
	ctx := context.Background()

	t.Setenv("MY_VAR", "foo")

	provider := New()
	v, err := provider.Get(ctx, "MY_VAR")
	assert.NoError(t, err)
	assert.Equal(t, "foo", v)

	v, err = provider.Get(ctx, "MISSING")
	assert.Error(t, err)
	assert.Empty(t, v)
}
