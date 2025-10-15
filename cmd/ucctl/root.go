package main

import (
	"github.com/spf13/cobra"

	"userclouds.com/cmd/ucctl/synctenant"
)

const (
	RootUsage       = "ucctl"
	RootShort       = "CLI utility for interacting with userclouds"
	RootLong        = `CLI utility for interacting with userclouds`
	SyncTenantUsage = "synctenant [ARG...]"
	SyncTenantShort = "Sync userclouds tenant resources"
	SyncTenantLong  = `Sync userclouds tenant resources`
)

type Root struct{}

func NewRoot() *Root {
	return &Root{}
}

func (r *Root) Execute() error {
	return r.Command().Execute()
}

func (r *Root) Command() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   RootUsage,
		Short: RootShort,
		Long:  RootLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(SyncTenantCommand())
	return rootCmd
}

func SyncTenantCommand() *cobra.Command {
	st := synctenant.Command{}
	cmd := &cobra.Command{
		Use:   SyncTenantUsage,
		Short: SyncTenantShort,
		Long:  SyncTenantLong,
		RunE:  st.RunE,
	}

	// TODO: Right now only authz is supported.  Add tokenizer, userstore, authn, and logserver.

	cmd.PersistentFlags().BoolVarP(&st.Verbose, "verbose", "v", false, "verbose output")
	cmd.PersistentFlags().StringVarP(&st.SourceURL, "source-url", "", "", "source URL")
	cmd.PersistentFlags().StringVarP(&st.SourceClientId, "source-client-id", "", "", "source client ID")
	cmd.PersistentFlags().StringVarP(&st.SourceClientSecretVar, "source-client-secret", "", synctenant.DefaultClientSecretVar, "source client secret")
	cmd.PersistentFlags().StringVarP(&st.DestinationURL, "destination-url", "", "", "destination URL")
	cmd.PersistentFlags().StringVarP(&st.DestinationClientId, "destination-client-id", "", "", "destination client id")
	cmd.PersistentFlags().StringVarP(&st.DestinationClientSecretVar, "destination-client-secret", "", synctenant.DefaultClientSecretVar, "destination client secret")
	cmd.PersistentFlags().BoolVarP(&st.DryRun, "dry-run", "", false, "dry run")
	cmd.PersistentFlags().BoolVarP(&st.InsertOnly, "insert-only", "", false, "only insert only")
	return cmd
}
