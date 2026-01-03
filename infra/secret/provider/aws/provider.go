package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

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

type Provider struct {
	client  Client
	region  string
	profile string
}

// Provider is a SecretProvider implementation for AWS resources.
// TODO: need to turn on multi-region replication for secret manager
// TODO: need to turn on secret rotation
// TODO: need to audit which creds have access to which secrets
func New() *Provider {
	return &Provider{}
}

// WithSecretsmanagerClient sets the client to
func (p *Provider) WithSecretsManagerClient(client Client) *Provider {
	p.client = client
	return p
}

func (p *Provider) Prefix() string {
	return Prefix
}

func (p *Provider) IsDev() bool {
	return false
}

// HasValidParams validates that only supported query parameters are present
// Supported parameters: profile, region
func (p *Provider) HasValidParams(params url.Values) error {
	supportedParams := map[string]bool{
		"profile": true,
		"region":  true,
	}

	for key := range params {
		if !supportedParams[key] {
			return fmt.Errorf("unsupported query parameter %q for AWS provider (supported: profile, region)", key)
		}
	}

	return nil
}

func (p *Provider) Get(ctx context.Context, path string) (string, error) {
	// Extract query params and get clean path
	cleanPath := p.extractAndSetParams(path)

	if err := p.initClient(ctx); err != nil {
		return "", ucerr.Wrap(err)
	}

	// VersionStage defaults to AWSCURRENT if unspecified
	input := &secretsmanager.GetSecretValueInput{SecretId: &cleanPath, VersionStage: aws.String("AWSCURRENT")}
	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
	result, err := p.client.GetSecretValue(ctx, input)
	if err != nil {
		return "", ucerr.Errorf("failed to load AWS secret '%s' from '%s': %w", cleanPath, p.region, err)
	}
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

func (p *Provider) Save(ctx context.Context, path, secret string) error {
	// Extract query params and get clean path
	cleanPath := p.extractAndSetParams(path)

	if err := p.initClient(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	// serialize the secret into our silly awsSecret JSON blob
	j, err := json.Marshal(awsSecret{secret})
	if err != nil {
		return ucerr.Wrap(err)
	}
	js := string(j)

	uclog.Infof(ctx, "creating secret '%s' in AWS", cleanPath)
	_, err = p.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{Name: &cleanPath, SecretString: &js, Tags: getTagsForSecret()})
	if err == nil {
		return nil
	}
	var resourceExistsErr *types.ResourceExistsException
	if errors.As(err, &resourceExistsErr) {
		uclog.Infof(ctx, "Secret '%s' already exists, updating it instead", cleanPath)
		_, err = p.client.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{SecretId: &cleanPath, SecretString: &js})
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(err)
}

func (p *Provider) Delete(ctx context.Context, path string) error {
	// Extract query params and get clean path
	cleanPath := p.extractAndSetParams(path)

	if err := p.initClient(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "Delete secret '%s' in AWS", cleanPath)
	_, err := p.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{SecretId: &cleanPath, RecoveryWindowInDays: aws.Int64(DefaultSecretRecoveryWindowInDays)})
	return ucerr.Wrap(err)
}

func (p *Provider) initClient(ctx context.Context) error {
	if p.client != nil {
		return nil
	}

	if p.profile != "" {
		return p.initClientWithProfile(ctx)
	}

	return p.initClientDefault(ctx)
}

func (p *Provider) initClientDefault(ctx context.Context) error {
	// TODO: This should respect p.region if set via query params, similar to initClientWithProfile.
	// Currently maintaining existing behavior to avoid breaking changes. Need to test and update.
	cfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	p.client = secretsmanager.NewFromConfig(cfg)
	p.region = cfg.Region

	return nil
}

func (p *Provider) initClientWithProfile(ctx context.Context) error {
	// Use region if specified, otherwise use default region
	region := p.region
	if region == "" {
		region = ucaws.DefaultRegion
	}

	cfg, err := ucaws.NewConfigForProfile(ctx, region, p.profile)
	if err != nil {
		return ucerr.Wrap(err)
	}

	p.client = secretsmanager.NewFromConfig(cfg)
	// Only set p.region if it wasn't already set from query params
	if p.region == "" {
		p.region = cfg.Region
	}

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

// extractAndSetParams extracts query parameters from the path
// and sets them on the provider, then returns the clean path without query params.
// Example: "my-secret?profile=production&region=us-east-1" -> sets p.profile="production", p.region="us-east-1", returns "my-secret"
// Note: Path should already have the prefix stripped (via ValueWithParams) before being passed here
func (p *Provider) extractAndSetParams(path string) string {
	// Check if there are query parameters
	idx := strings.Index(path, "?")
	if idx == -1 {
		// No query parameters, return as-is
		return path
	}

	// Split path and query string
	cleanPath := path[:idx]
	queryString := path[idx+1:]

	// Parse query parameters
	params, err := url.ParseQuery(queryString)
	if err != nil {
		// If parsing fails, just return the original path
		return path
	}

	// Set profile if provided in query params
	if profile := params.Get("profile"); profile != "" {
		p.profile = profile
	}

	// Set region if provided in query params
	if region := params.Get("region"); region != "" {
		p.region = region
	}

	return cleanPath
}
