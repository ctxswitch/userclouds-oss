package get

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

type ObjectsCommand struct {
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

func (c *ObjectsCommand) RunE(cmd *cobra.Command, args []string) error {
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

	config := common.PagerConfig[authz.Object]{
		Ctx: cmd.Context(),
		FetchFunc: func(ctx context.Context, cursor string, limit int) ([]authz.Object, string, error) {
			return c.fetchObjects(ctx, azClient, cursor, limit)
		},
		TableRenderFunc: c.renderTable,
		InitialCursor:   c.Cursor,
		NoItemsMessage:  "No objects found.",
		ItemName:        "objects",
	}

	return common.RunPager(config)
}

func (c *ObjectsCommand) outputFormatted(ctx context.Context, azClient *authz.Client) error {
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
		objects, nextCursor, err := c.fetchObjects(ctx, azClient, cursor, 100)
		if err != nil {
			return err
		}

		if len(objects) == 0 {
			break
		}

		// Write this batch immediately
		switch format {
		case common.OutputFormatJSON:
			if err := writer.(*common.JSONStreamWriter).WriteItems(objects); err != nil {
				return err
			}
		case common.OutputFormatYAML:
			if err := writer.(*common.YAMLStreamWriter).WriteItems(objects); err != nil {
				return err
			}
		case common.OutputFormatCSV:
			if err := writer.(*common.CSVWriter).WriteItems(objects); err != nil {
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

func (c *ObjectsCommand) fetchObjects(ctx context.Context, azClient *authz.Client, cursor string, limit int) ([]authz.Object, string, error) {
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
		filterStr, err := common.FormatFilterString(c.Filter, common.ObjectFilterKeys)
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
	resp, err := azClient.ListObjects(ctx, opts...)
	if err != nil {
		return nil, "", ucerr.Wrap(err)
	}

	nextCursor := ""
	if resp.HasNext {
		nextCursor = string(resp.Next)
	}

	return resp.Data, nextCursor, nil
}

func (c *ObjectsCommand) renderTable(objects []authz.Object) ([]common.TableColumn, []common.TableRow) {
	columns := []common.TableColumn{
		{Header: "ID", Width: 36},                // UUIDs are always 36 characters
		{Header: "ALIAS", Width: 0},              // Dynamic width for aliases
		{Header: "TYPE_ID", Width: 36},           // UUIDs are always 36 characters
		{Header: "ORGANIZATION_ID", Width: 36},   // UUIDs are always 36 characters
		{Header: "CREATED", Width: 19},           // "2006-01-02 15:04:05" is 19 characters
		{Header: "UPDATED", Width: 19},           // "2006-01-02 15:04:05" is 19 characters
	}

	rows := make([]common.TableRow, len(objects))
	for i, obj := range objects {
		var alias string
		if obj.Alias != nil {
			alias = *obj.Alias
		}
		rows[i] = common.TableRow{
			obj.ID.String(),
			alias,
			obj.TypeID.String(),
			obj.OrganizationID.String(),
			obj.Created.Format("2006-01-02 15:04:05"),
			obj.Updated.Format("2006-01-02 15:04:05"),
		}
	}

	return columns, rows
}
