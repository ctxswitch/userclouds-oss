package get

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

type EdgeTypesCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	Limit           int
	Cursor          string
	NoPager         bool
	Filter          string
	RawFilter       string
	Output          string

	credentials *common.Credentials
}

func (c *EdgeTypesCommand) RunE(cmd *cobra.Command, args []string) error {
	// Validate output format if specified
	if c.Output != "" && c.Output != "table" {
		if err := common.ValidateOutputFormat(c.Output); err != nil {
			return err
		}
	}

	// Load credentials from context or flags
	creds, err := common.LoadCredentialsFromContext(
		
		
		c.URL,
		c.ClientID,
		c.ClientSecret,
		c.ClientSecretVar,
		"", // configPath - use default precedence
	)
	if err != nil {
		return err
	}
	c.credentials = creds

	// Create client credentials option
	credOpt, err := c.credentials.GetClientCredentials()
	if err != nil {
		return fmt.Errorf("failed to create client credentials: %w", err)
	}

	// Create AuthZ client
	azClient, err := authz.NewClient(c.credentials.URL, authz.JSONClient(credOpt))
	if err != nil {
		return fmt.Errorf("failed to create AuthZ client: %w", err)
	}

	// If output format is specified (and not table), fetch all data and output in that format
	if c.Output != "" && c.Output != "table" {
		return c.outputFormatted(cmd.Context(), azClient)
	}

	// Check if we should use the pager
	usePager := !c.NoPager && term.IsTerminal(int(os.Stdout.Fd()))

	config := common.PagerConfig[authz.EdgeType]{
		Ctx: cmd.Context(),
		FetchFunc: func(ctx context.Context, cursor string, limit int) ([]authz.EdgeType, string, error) {
			return c.fetchEdgeTypes(ctx, azClient, cursor, limit)
		},
		DisplayFunc:          c.displayWithHeader,
		DisplayWithoutHeader: c.displayWithoutHeader,
		InitialCursor:        c.Cursor,
		NoItemsMessage:       "No edge types found.",
		ItemName:             "edge types",
	}

	if usePager {
		return common.RunInteractivePager(config)
	}

	return common.RunNonInteractivePager(config)
}

func (c *EdgeTypesCommand) outputFormatted(ctx context.Context, azClient *authz.Client) error {
	cursor := c.Cursor
	format := common.OutputFormat(c.Output)

	// Create appropriate streaming writer
	var writer interface{}
	switch format {
	case common.OutputFormatJSON:
		writer = common.NewJSONStreamWriter()
	case common.OutputFormatYAML:
		w := common.NewYAMLStreamWriter()
		defer w.Close()
		writer = w
	case common.OutputFormatCSV:
		writer = common.NewCSVWriter()
	default:
		return fmt.Errorf("unsupported output format: %s", c.Output)
	}

	// Stream results as they come in
	for {
		edgeTypes, nextCursor, err := c.fetchEdgeTypes(ctx, azClient, cursor, 100)
		if err != nil {
			return err
		}

		if len(edgeTypes) == 0 {
			break
		}

		// Write this batch immediately
		switch format {
		case common.OutputFormatJSON:
			if err := writer.(*common.JSONStreamWriter).WriteItems(edgeTypes); err != nil {
				return err
			}
		case common.OutputFormatYAML:
			if err := writer.(*common.YAMLStreamWriter).WriteItems(edgeTypes); err != nil {
				return err
			}
		case common.OutputFormatCSV:
			if err := writer.(*common.CSVWriter).WriteItems(edgeTypes); err != nil {
				return err
			}
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return nil
}

func (c *EdgeTypesCommand) fetchEdgeTypes(ctx context.Context, azClient *authz.Client, cursor string, limit int) ([]authz.EdgeType, string, error) {
	var paginationOpts []pagination.Option

	// Validate that both filter and raw-filter are not specified
	if c.Filter != "" && c.RawFilter != "" {
		return nil, "", fmt.Errorf("cannot specify both --filter and --raw-filter")
	}

	// Add filter if specified
	if c.RawFilter != "" {
		// Use raw filter directly
		paginationOpts = append(paginationOpts, pagination.Filter(c.RawFilter))
	} else if c.Filter != "" {
		// Parse and format the filter
		filterStr, err := common.FormatFilterString(c.Filter, common.EdgeTypeFilterKeys)
		if err != nil {
			return nil, "", err
		}
		paginationOpts = append(paginationOpts, pagination.Filter(filterStr))
	}

	// Add cursor and limit
	if cursor != "" {
		paginationOpts = append(paginationOpts, pagination.StartingAfter(pagination.Cursor(cursor)))
	}
	if limit > 0 {
		paginationOpts = append(paginationOpts, pagination.Limit(limit))
	}

	// Build authz options
	var opts []authz.Option
	if len(paginationOpts) > 0 {
		opts = append(opts, authz.Pagination(paginationOpts...))
	}

	// Use paginated endpoint
	resp, err := azClient.ListEdgeTypesPaginated(ctx, opts...)
	if err != nil {
		return nil, "", ucerr.Wrap(err)
	}

	nextCursor := ""
	if resp.HasNext {
		nextCursor = string(resp.Next)
	}

	return resp.Data, nextCursor, nil
}

func (c *EdgeTypesCommand) displayWithHeader(edgeTypes []authz.EdgeType) {
	display := common.NewTabularDisplay()
	display.WriteHeader("ID", "TYPE_NAME", "SOURCE_TYPE_ID", "TARGET_TYPE_ID", "ORGANIZATION_ID", "CREATED", "UPDATED")
	for _, et := range edgeTypes {
		display.WriteRow(
			et.ID.String(),
			et.TypeName,
			et.SourceObjectTypeID.String(),
			et.TargetObjectTypeID.String(),
			et.OrganizationID.String(),
			et.Created.Format("2006-01-02 15:04:05"),
			et.Updated.Format("2006-01-02 15:04:05"),
		)
	}
	display.Flush()
}

func (c *EdgeTypesCommand) displayWithoutHeader(edgeTypes []authz.EdgeType) {
	display := common.NewTabularDisplay()
	for _, et := range edgeTypes {
		display.WriteRow(
			et.ID.String(),
			et.TypeName,
			et.SourceObjectTypeID.String(),
			et.TargetObjectTypeID.String(),
			et.OrganizationID.String(),
			et.Created.Format("2006-01-02 15:04:05"),
			et.Updated.Format("2006-01-02 15:04:05"),
		)
	}
	display.Flush()
}
