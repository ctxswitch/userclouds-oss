package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/mock"

	"testing"

	"github.com/stretchr/testify/assert"
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
