package aws

// AWS insists on returning a JSON blob of key/value instead of just a string
// if/when we implement eg. GCP's equivalent, or Hashicorp, then some of this
// could get moved over to aws.go and UnmarshalYAML could have a "provider" interface
type awsSecret struct {
	String string `json:"string" yaml:"string"`
}
