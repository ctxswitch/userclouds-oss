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

type ObjectTypesCommand struct {
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

func (c *ObjectTypesCommand) RunE(cmd *cobra.Command, args []string) error {
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

	config := common.PagerConfig[authz.ObjectType]{
		Ctx: cmd.Context(),
		FetchFunc: func(ctx context.Context, cursor string, limit int) ([]authz.ObjectType, string, error) {
			return c.fetchObjectTypes(ctx, azClient, cursor, limit)
		},
		DisplayFunc:          c.displayWithHeader,
		DisplayWithoutHeader: c.displayWithoutHeader,
		InitialCursor:        c.Cursor,
		NoItemsMessage:       "No object types found.",
		ItemName:             "object types",
	}

	if usePager {
		return common.RunInteractivePager(config)
	}

	return common.RunNonInteractivePager(config)
}

func (c *ObjectTypesCommand) outputFormatted(ctx context.Context, azClient *authz.Client) error {
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
		objectTypes, nextCursor, err := c.fetchObjectTypes(ctx, azClient, cursor, 100)
		if err != nil {
			return err
		}

		if len(objectTypes) == 0 {
			break
		}

		// Write this batch immediately
		switch format {
		case common.OutputFormatJSON:
			if err := writer.(*common.JSONStreamWriter).WriteItems(objectTypes); err != nil {
				return err
			}
		case common.OutputFormatYAML:
			if err := writer.(*common.YAMLStreamWriter).WriteItems(objectTypes); err != nil {
				return err
			}
		case common.OutputFormatCSV:
			if err := writer.(*common.CSVWriter).WriteItems(objectTypes); err != nil {
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

func (c *ObjectTypesCommand) fetchObjectTypes(ctx context.Context, azClient *authz.Client, cursor string, limit int) ([]authz.ObjectType, string, error) {
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
		filterStr, err := common.FormatFilterString(c.Filter, common.ObjectTypeFilterKeys)
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
	resp, err := azClient.ListObjectTypesPaginated(ctx, opts...)
	if err != nil {
		return nil, "", ucerr.Wrap(err)
	}

	nextCursor := ""
	if resp.HasNext {
		nextCursor = string(resp.Next)
	}

	return resp.Data, nextCursor, nil
}

func (c *ObjectTypesCommand) displayWithHeader(objectTypes []authz.ObjectType) {
	display := common.NewTabularDisplay()
	display.WriteHeader("ID", "TYPE_NAME", "CREATED", "UPDATED")
	for _, ot := range objectTypes {
		display.WriteRow(
			ot.ID.String(),
			ot.TypeName,
			ot.Created.Format("2006-01-02 15:04:05"),
			ot.Updated.Format("2006-01-02 15:04:05"),
		)
	}
	display.Flush()
}

func (c *ObjectTypesCommand) displayWithoutHeader(objectTypes []authz.ObjectType) {
	display := common.NewTabularDisplay()
	for _, ot := range objectTypes {
		display.WriteRow(
			ot.ID.String(),
			ot.TypeName,
			ot.Created.Format("2006-01-02 15:04:05"),
			ot.Updated.Format("2006-01-02 15:04:05"),
		)
	}
	display.Flush()
}
