package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/mock"
)

type Client interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
	DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error)
}

type MockSecretsManagerClient struct {
	*secretsmanager.Client
	mock.Mock
}

func (c *MockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	args := c.Called(ctx, params, opts)
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

func (c *MockSecretsManagerClient) CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	args := c.Called(ctx, params, opts)
	return args.Get(0).(*secretsmanager.CreateSecretOutput), args.Error(1)
}

func (c *MockSecretsManagerClient) UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	args := c.Called(ctx, params, opts)
	return args.Get(0).(*secretsmanager.UpdateSecretOutput), args.Error(1)
}

func (c *MockSecretsManagerClient) DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error) {
	args := c.Called(ctx, params, opts)
	return args.Get(0).(*secretsmanager.DeleteSecretOutput), args.Error(1)
}
