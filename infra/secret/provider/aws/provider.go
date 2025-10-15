package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	Prefix                            = "aws://secrets/"
	DefaultSecretRecoveryWindowInDays = 7
)

// Provider is a SecretProvider implementation for AWS resources.
type Provider struct {
	client Client
	region string
}

// New returns an initialized provider.
// TODO: need to turn on multi-region replication for secret manager
// TODO: need to turn on secret rotation
// TODO: need to audit which creds have access to which secrets
func New() *Provider {
	return &Provider{}
}

// WithSecretsManagerClient overrides the client.  This is generally used
// for testing purposes.
func (p *Provider) WithSecretsManagerClient(client Client) *Provider {
	p.client = client
	return p
}

// Prefix returns the URI prefix for a secret stored in the AWS secrets manager.
func (p *Provider) Prefix() string {
	return Prefix
}

// IsDev is a helper function that returns true if the provider is explicitly used
// in development environments.  This allows for dev specific behaviors to be handled.
func (p *Provider) IsDev() bool {
	return false
}

// Get retrieves a secret version from a secret manager object and returns the value.
func (p *Provider) Get(ctx context.Context, path string) (string, error) {
	if err := p.initClient(ctx); err != nil {
		return "", ucerr.Wrap(err)
	}

	// VersionStage defaults to AWSCURRENT if unspecified
	input := &secretsmanager.GetSecretValueInput{SecretId: &path, VersionStage: aws.String("AWSCURRENT")}
	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
	result, err := p.client.GetSecretValue(ctx, input)
	if err != nil {
		return "", ucerr.Errorf("failed to load AWS secret '%s' from '%s': %w", path, p.region, err)
	}
	uclog.Debugf(ctx, "Loaded AWS secret '%s' from '%s'", path, p.region)
	value, err := decodeSecret(result)

	// decode AWS's JSON wrapper if necessary
	var awsSec awsSecret
	var secret string
	if err := json.Unmarshal([]byte(value), &awsSec); err == nil {
		secret = awsSec.String
	} else {
		secret = value
	}

	return secret, ucerr.Wrap(err)
}

// Save creates or updates a secret in the AWS secrets manager.
func (p *Provider) Save(ctx context.Context, path, secret string) error {
	if err := p.initClient(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	// serialize the secret into our silly awsSecret JSON blob
	j, err := json.Marshal(awsSecret{secret})
	if err != nil {
		return ucerr.Wrap(err)
	}
	js := string(j)

	uclog.Infof(ctx, "creating secret '%s' in AWS", path)
	_, err = p.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{Name: &path, SecretString: &js, Tags: getTagsForSecret()})
	if err == nil {
		return nil
	}
	var resourceExistsErr *types.ResourceExistsException
	if errors.As(err, &resourceExistsErr) {
		uclog.Infof(ctx, "Secret '%s' already exists, updating it instead", path)
		_, err = p.client.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{SecretId: &path, SecretString: &js})
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(err)
}

// Delete removes a secret from the AWS secrets manager.
func (p *Provider) Delete(ctx context.Context, path string) error {
	if err := p.initClient(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "Delete secret '%s' in AWS", path)
	_, err := p.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{SecretId: &path, RecoveryWindowInDays: aws.Int64(DefaultSecretRecoveryWindowInDays)})
	return ucerr.Wrap(err)
}

// initClient is a helper that initializes the AWS client.
func (p *Provider) initClient(ctx context.Context) error {
	if p.client != nil {
		return nil
	}

	cfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	p.client = secretsmanager.NewFromConfig(cfg)
	p.region = cfg.Region

	return nil
}

func decodeSecret(result *secretsmanager.GetSecretValueOutput) (string, error) {
	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secret string
	if result.SecretString != nil {
		secret = *result.SecretString
	} else {
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		length, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			return "", ucerr.Wrap(err)
		}
		secret = string(decodedBinarySecretBytes[:length])
	}
	if secret == "" {
		return "", ucerr.Errorf("failed to decode secret %s", *result.Name)
	}
	return secret, nil
}

func getTagsForSecret() []types.Tag {
	uv := universe.Current()
	tags := []types.Tag{
		{
			Key:   aws.String(universe.EnvKeyUniverse),
			Value: aws.String(string(uv)),
		},
	}
	if uv.IsCloud() {
		tags = append(tags, types.Tag{
			Key:   aws.String("UC_ENV_TYPE"),
			Value: aws.String("eks"),
		})
	}
	return tags
}
