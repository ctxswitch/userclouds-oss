package common

import "github.com/spf13/cobra"

// AddFilterFlags adds standard filter flags to a command
func AddFilterFlags(cmd *cobra.Command, filter, rawFilter *string) {
	cmd.Flags().StringVarP(filter, "filter", "f", "", "filter results (use & for AND, | for OR, () for grouping; e.g., type_name=user&id=123)")
	cmd.Flags().StringVarP(rawFilter, "raw-filter", "", "", "raw filter query (see docs for format; mutually exclusive with --filter)")
}

// AddPaginationFlags adds standard pagination flags to a command
func AddPaginationFlags(cmd *cobra.Command, limit *int, cursor *string, noPager *bool) {
	cmd.Flags().IntVarP(limit, "limit", "l", 0, "maximum number of results to return (0 = terminal height)")
	cmd.Flags().StringVarP(cursor, "cursor", "c", "", "pagination cursor for next page")
	cmd.Flags().BoolVarP(noPager, "no-pager", "", false, "disable interactive paging (show all results)")
}

// AddOutputFlag adds standard output format flag to a command
func AddOutputFlag(cmd *cobra.Command, output *string) {
	cmd.Flags().StringVarP(output, "output", "o", "table", "output format: table, json, yaml, csv (disables pager)")
}

// AddListCommandFlags adds all standard flags for list commands (auth, pagination, filter, output)
// If authnType is nil, the authn-type flag is not added
func AddListCommandFlags(cmd *cobra.Command, url, clientID, clientSecret, clientSecretVar *string, limit *int, cursor *string, noPager *bool, filter, rawFilter, output *string, authnType *string) {
	AddAuthFlags(cmd, url, clientID, clientSecret, clientSecretVar, authnType)
	AddPaginationFlags(cmd, limit, cursor, noPager)
	AddFilterFlags(cmd, filter, rawFilter)
	AddOutputFlag(cmd, output)
}
