package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAWS_getAWSSecretWithClient(t *testing.T) {
	ctx := context.Background()
	sm := &MockSecretsManagerClient{}
	sm.On("GetSecretValue", ctx, mock.Anything, mock.Anything).Return(&secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(`{"string":"testsecret"}`),
	}, nil)

	provider := New().WithSecretsManagerClient(sm)
	secret, err := provider.Get(ctx, "dummysecret")
	assert.NoError(t, err)
	assert.Equal(t, "testsecret", secret)
}

func TestAWS_HasValidParams(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string][]string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "no params is valid",
			params:      map[string][]string{},
			expectError: false,
		},
		{
			name: "profile param is valid",
			params: map[string][]string{
				"profile": {"production"},
			},
			expectError: false,
		},
		{
			name: "region param is valid",
			params: map[string][]string{
				"region": {"us-west-2"},
			},
			expectError: false,
		},
		{
			name: "both profile and region params are valid",
			params: map[string][]string{
				"profile": {"production"},
				"region":  {"us-east-1"},
			},
			expectError: false,
		},
		{
			name: "unsupported param fails",
			params: map[string][]string{
				"invalid": {"value"},
			},
			expectError: true,
			errorMsg:    "unsupported query parameter \"invalid\" for AWS provider",
		},
		{
			name: "multiple unsupported params fails on first",
			params: map[string][]string{
				"invalid":  {"value"},
				"invalid2": {"value2"},
			},
			expectError: true,
		},
		{
			name: "mixed supported and unsupported params fails",
			params: map[string][]string{
				"profile": {"production"},
				"region":  {"us-west-2"},
				"invalid": {"value"},
			},
			expectError: true,
			errorMsg:    "unsupported query parameter \"invalid\" for AWS provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := New()
			err := provider.HasValidParams(tt.params)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
