package synctenant

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
)

const (
	DefaultClientSecretVar = "UC_CLIENT_SECRET"
)

type Command struct {
	SourceURL                  string
	SourceClientId             string
	SourceClientSecretVar      string
	DestinationURL             string
	DestinationClientId        string
	DestinationClientSecretVar string
	DryRun                     bool
	Verbose                    bool
	InsertOnly                 bool
}

func (c *Command) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	logLevel := uclog.LogLevelInfo
	if c.Verbose {
		logLevel = uclog.LogLevelDebug
	}

	logtransports.InitLoggerAndTransportsForTools(ctx, logLevel, logLevel, "synctenant")
	defer logtransports.Close()

	if err := c.validate(); err != nil {
		uclog.Errorf(ctx, err.Error())
		os.Exit(1)
	}

	if err := c.sync(ctx); err != nil {
		uclog.Errorf(ctx, err.Error())
		os.Exit(1)
	}

	return nil
}

func (c *Command) sync(ctx context.Context) error {
	startTime := time.Now().UTC()
	defer func() {
		endTime := time.Now().UTC()
		duration := endTime.Sub(startTime)
		uclog.Infof(ctx, "synctenant took %s", duration)
	}()

	uclog.Infof(ctx, "Fetching: %s", c.SourceURL)
	srcTenant := NewTenant(c.SourceURL, c.SourceClientId, c.SourceClientSecretVar)
	srcClient, err := srcTenant.GetClient()
	if err != nil {
		return fmt.Errorf("failed to create tenant %s: %v", c.SourceURL, err)
	}
	srcResources := NewResources()
	if err := srcResources.Get(ctx, srcClient); err != nil {
		return fmt.Errorf("failed to get resources from %s: %v", c.SourceURL, err)
	}

	uclog.Infof(ctx, "Fetching: %s", c.DestinationURL)
	dstTenant := NewTenant(c.DestinationURL, c.DestinationClientId, c.DestinationClientSecretVar)
	dstClient, err := dstTenant.GetClient()
	if err != nil {
		return fmt.Errorf("failed to create tenant %s: %v", c.DestinationClientId, err)
	}
	dstResources := NewResources()
	if err := dstResources.Get(ctx, dstClient); err != nil {
		return fmt.Errorf("failed to get resources from %s: %v", c.DestinationURL, err)
	}

	if !c.InsertOnly {
		uclog.Infof(ctx, "Determining deletions")
		deleteResources := NewResources()
		deleteResources.Diff(ctx, dstResources, srcResources)

		if !c.DryRun {
			if err := deleteResources.Delete(ctx, dstClient); err != nil {
				return fmt.Errorf("failed to delete resources from %s: %v", c.DestinationURL, err)
			}
		} else {
			uclog.Infof(ctx, "Dryrun enabled, skipping deletion")
		}
	} else {
		uclog.Infof(ctx, "Insert only has been requested, skipping deletions")
	}

	uclog.Infof(ctx, "Determining insertions")
	insertResources := NewResources()
	insertResources.Diff(ctx, srcResources, dstResources)

	if c.DryRun {
		uclog.Infof(ctx, "DryRun enabled, skipping insertions")
		return nil
	}

	err = insertResources.Insert(ctx, dstClient)
	if err != nil {
		return fmt.Errorf("failed to insert resources from %s: %v", c.DestinationURL, err)
	}

	return nil
}

func (c *Command) validate() error {
	var err error
	if c.SourceURL == "" {
		return fmt.Errorf("source URL is required")
	}

	if c.SourceClientId == "" {
		return fmt.Errorf("source client id is required")
	}

	if os.Getenv(c.SourceClientSecretVar) == "" {
		return fmt.Errorf("source client secret is not set")
	}

	if c.DestinationURL == "" {
		return fmt.Errorf("destination URL is required")
	}

	if c.DestinationClientId == "" {
		return fmt.Errorf("destination client id is required")
	}

	if os.Getenv(c.DestinationClientSecretVar) == "" {
		return fmt.Errorf("destination client secret is not set")
	}

	return err
}
