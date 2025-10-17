package sync

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/config"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

const (
	DefaultClientSecretVar = "UC_CLIENT_SECRET"
)

// TenantCommand handles syncing tenant resources between environments
type TenantCommand struct {
	Source                     string
	SourceURL                  string
	SourceClientId             string
	SourceClientSecretVar      string
	Destination                string
	DestinationURL             string
	DestinationClientId        string
	DestinationClientSecretVar string
	DryRun                     bool
	Verbose                    bool
	InsertOnly                 bool
}

// NewTenantCommand creates a new tenant sync command
func NewTenantCommand() *cobra.Command {
	tc := &TenantCommand{}
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Sync tenant resources between environments",
		Long: `Sync UserClouds tenant resources (currently supports AuthZ resources) from a source tenant to a destination tenant.

This command fetches resources from the source tenant and applies them to the destination tenant.
It supports dry-run mode to preview changes and insert-only mode to avoid deletions.`,
		RunE: tc.RunE,
	}

	cmd.Flags().BoolVarP(&tc.Verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&tc.Source, "source", "", "", "source context name (alternative to --source-url/--source-client-id)")
	cmd.Flags().StringVarP(&tc.SourceURL, "source-url", "", "", "source tenant URL")
	cmd.Flags().StringVarP(&tc.SourceClientId, "source-client-id", "", "", "source client ID")
	cmd.Flags().StringVarP(&tc.SourceClientSecretVar, "source-client-secret-var", "", DefaultClientSecretVar, "environment variable containing source client secret")
	cmd.Flags().StringVarP(&tc.Destination, "destination", "", "", "destination context name (alternative to --destination-url/--destination-client-id)")
	cmd.Flags().StringVarP(&tc.DestinationURL, "destination-url", "", "", "destination tenant URL")
	cmd.Flags().StringVarP(&tc.DestinationClientId, "destination-client-id", "", "", "destination client ID")
	cmd.Flags().StringVarP(&tc.DestinationClientSecretVar, "destination-client-secret-var", "", DefaultClientSecretVar, "environment variable containing destination client secret")
	cmd.Flags().BoolVarP(&tc.DryRun, "dry-run", "", false, "preview changes without applying them")
	cmd.Flags().BoolVarP(&tc.InsertOnly, "insert-only", "", false, "only insert new resources, don't delete existing ones")

	return cmd
}

func (c *TenantCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if err := c.validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := c.sync(ctx); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	return nil
}

func (c *TenantCommand) validate() error {
	// Load config if either source or destination context is specified
	var cfg *config.Config
	if c.Source != "" || c.Destination != "" {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Handle source configuration
	if c.Source != "" {
		ctx, err := cfg.GetContext(c.Source)
		if err != nil {
			return fmt.Errorf("failed to get source context: %w", err)
		}
		// Only override if not explicitly set
		if c.SourceURL == "" {
			c.SourceURL = ctx.URL
		}
		if c.SourceClientId == "" {
			c.SourceClientId = ctx.ClientID
		}
		// Set client secret from context in environment variable if not already set
		if os.Getenv(c.SourceClientSecretVar) == "" {
			if ctx.ClientSecret != "" {
				os.Setenv(c.SourceClientSecretVar, ctx.ClientSecret)
			}
		}
	}

	// Handle destination configuration
	if c.Destination != "" {
		ctx, err := cfg.GetContext(c.Destination)
		if err != nil {
			return fmt.Errorf("failed to get destination context: %w", err)
		}
		// Only override if not explicitly set
		if c.DestinationURL == "" {
			c.DestinationURL = ctx.URL
		}
		if c.DestinationClientId == "" {
			c.DestinationClientId = ctx.ClientID
		}
		// Set client secret from context in environment variable if not already set
		if os.Getenv(c.DestinationClientSecretVar) == "" {
			if ctx.ClientSecret != "" {
				os.Setenv(c.DestinationClientSecretVar, ctx.ClientSecret)
			}
		}
	}

	// Validate that all required fields are now set
	if c.SourceURL == "" {
		return fmt.Errorf("source URL is required (use --source or --source-url)")
	}

	if c.SourceClientId == "" {
		return fmt.Errorf("source client ID is required (use --source or --source-client-id)")
	}

	if os.Getenv(c.SourceClientSecretVar) == "" {
		return fmt.Errorf("source client secret is not set in environment variable %s", c.SourceClientSecretVar)
	}

	if c.DestinationURL == "" {
		return fmt.Errorf("destination URL is required (use --destination or --destination-url)")
	}

	if c.DestinationClientId == "" {
		return fmt.Errorf("destination client ID is required (use --destination or --destination-client-id)")
	}

	if os.Getenv(c.DestinationClientSecretVar) == "" {
		return fmt.Errorf("destination client secret is not set in environment variable %s", c.DestinationClientSecretVar)
	}

	return nil
}

func (c *TenantCommand) sync(ctx context.Context) error {
	startTime := time.Now().UTC()

	pterm.DefaultHeader.WithFullWidth().Println("Tenant Sync")
	pterm.Println()

	// Fetch source resources
	spinner, _ := pterm.DefaultSpinner.Start("Fetching resources from source: " + c.SourceURL)
	srcTenant := newTenant(c.SourceURL, c.SourceClientId, c.SourceClientSecretVar)
	srcClient, err := srcTenant.getClient()
	if err != nil {
		spinner.Fail("Failed to create source client")
		return fmt.Errorf("failed to create source client: %w", err)
	}
	srcResources := newResources()
	if err := srcResources.fetch(ctx, srcClient, spinner, c.Verbose); err != nil {
		spinner.Fail("Failed to fetch source resources")
		return fmt.Errorf("failed to fetch source resources: %w", err)
	}
	spinner.Success(fmt.Sprintf("Fetched source resources (%d object types, %d objects, %d edge types, %d edges)",
		len(srcResources.objectTypes), len(srcResources.objects), len(srcResources.edgeTypes), len(srcResources.edges)))

	// Fetch destination resources
	spinner, _ = pterm.DefaultSpinner.Start("Fetching resources from destination: " + c.DestinationURL)
	dstTenant := newTenant(c.DestinationURL, c.DestinationClientId, c.DestinationClientSecretVar)
	dstClient, err := dstTenant.getClient()
	if err != nil {
		spinner.Fail("Failed to create destination client")
		return fmt.Errorf("failed to create destination client: %w", err)
	}
	dstResources := newResources()
	if err := dstResources.fetch(ctx, dstClient, spinner, c.Verbose); err != nil {
		spinner.Fail("Failed to fetch destination resources")
		return fmt.Errorf("failed to fetch destination resources: %w", err)
	}
	spinner.Success(fmt.Sprintf("Fetched destination resources (%d object types, %d objects, %d edge types, %d edges)",
		len(dstResources.objectTypes), len(dstResources.objects), len(dstResources.edgeTypes), len(dstResources.edges)))

	// Handle deletions
	if !c.InsertOnly {
		spinner, _ = pterm.DefaultSpinner.Start("Computing resources to delete...")
		deleteResources := newResources()
		deleteResources.diff(ctx, dstResources, srcResources, c.Verbose)
		spinner.Success(fmt.Sprintf("Computed deletions (%d object types, %d objects, %d edge types, %d edges)",
			len(deleteResources.objectTypes), len(deleteResources.objects), len(deleteResources.edgeTypes), len(deleteResources.edges)))

		if !c.DryRun {
			if len(deleteResources.objectTypes) > 0 || len(deleteResources.objects) > 0 ||
				len(deleteResources.edgeTypes) > 0 || len(deleteResources.edges) > 0 {
				if err := deleteResources.delete(ctx, dstClient, c.Verbose); err != nil {
					return fmt.Errorf("failed to delete resources: %w", err)
				}
			}
		} else {
			pterm.Info.Println("Dry-run mode: skipping deletions")
		}
	} else {
		pterm.Info.Println("Insert-only mode: skipping deletions")
	}

	// Handle insertions
	spinner, _ = pterm.DefaultSpinner.Start("Computing resources to insert...")
	insertResources := newResources()
	insertResources.diff(ctx, srcResources, dstResources, c.Verbose)
	spinner.Success(fmt.Sprintf("Computed insertions (%d object types, %d objects, %d edge types, %d edges)",
		len(insertResources.objectTypes), len(insertResources.objects), len(insertResources.edgeTypes), len(insertResources.edges)))

	if c.DryRun {
		pterm.Info.Println("Dry-run mode: skipping insertions")
		duration := time.Since(startTime)
		pterm.Println()
		pterm.Success.Printfln("Sync completed in %s (dry-run)", duration)
		return nil
	}

	if len(insertResources.objectTypes) > 0 || len(insertResources.objects) > 0 ||
		len(insertResources.edgeTypes) > 0 || len(insertResources.edges) > 0 {
		if err := insertResources.insert(ctx, dstClient, c.Verbose); err != nil {
			return fmt.Errorf("failed to insert resources: %w", err)
		}
	}

	duration := time.Since(startTime)
	pterm.Println()
	pterm.Success.Printfln("Sync completed in %s", duration)
	return nil
}

// tenant represents a UserClouds tenant with authentication
type tenant struct {
	tenantURL       string
	clientID        string
	clientSecretVar string
	tokenSource     jsonclient.Option
}

func newTenant(tenantURL, clientID, clientSecretVar string) *tenant {
	return &tenant{
		tenantURL:       tenantURL,
		clientID:        clientID,
		clientSecretVar: clientSecretVar,
	}
}

func (t *tenant) getClient() (*authz.Client, error) {
	if err := t.initToken(); err != nil {
		return nil, err
	}
	return authz.NewClient(t.tenantURL, authz.JSONClient(t.tokenSource))
}

func (t *tenant) initToken() error {
	if _, err := url.Parse(t.tenantURL); err != nil {
		return fmt.Errorf("invalid tenant URL %s: %w", t.tenantURL, err)
	}

	secret := os.Getenv(t.clientSecretVar)
	if secret == "" {
		return fmt.Errorf("client secret not found in environment variable %s", t.clientSecretVar)
	}

	ts, err := jsonclient.ClientCredentialsForURL(t.tenantURL, t.clientID, secret, nil)
	if err != nil {
		return fmt.Errorf("failed to create token source: %w", err)
	}

	t.tokenSource = ts
	return nil
}

// resources holds collections of AuthZ resources
type resources struct {
	edgeTypes   []authz.EdgeType
	edges       []authz.Edge
	objectTypes []authz.ObjectType
	objects     []authz.Object
}

func newResources() *resources {
	return &resources{
		edgeTypes:   make([]authz.EdgeType, 0),
		edges:       make([]authz.Edge, 0),
		objectTypes: make([]authz.ObjectType, 0),
		objects:     make([]authz.Object, 0),
	}
}

func (r *resources) fetch(ctx context.Context, azc *authz.Client, spinner *pterm.SpinnerPrinter, verbose bool) error {
	if verbose {
		spinner.UpdateText("Fetching object types...")
	}
	if err := r.fetchObjectTypes(ctx, azc); err != nil {
		return err
	}

	if verbose {
		spinner.UpdateText("Fetching objects...")
	}
	if err := r.fetchObjects(ctx, azc); err != nil {
		return err
	}

	if verbose {
		spinner.UpdateText("Fetching edge types...")
	}
	if err := r.fetchEdgeTypes(ctx, azc); err != nil {
		return err
	}

	if verbose {
		spinner.UpdateText("Fetching edges...")
	}
	if err := r.fetchEdges(ctx, azc); err != nil {
		return err
	}

	return nil
}

func (r *resources) insert(ctx context.Context, azc *authz.Client, verbose bool) error {
	// Insert in dependency order: ObjectTypes -> Objects -> EdgeTypes -> Edges

	if len(r.objectTypes) > 0 {
		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Inserting %d object types...", len(r.objectTypes)))
		for _, ot := range r.objectTypes {
			if verbose {
				spinner.UpdateText(fmt.Sprintf("Inserting object type: %s", ot.TypeName))
			}
			if _, err := azc.CreateObjectType(ctx, ot.ID, ot.TypeName); err != nil {
				spinner.Fail(fmt.Sprintf("Failed to insert object type: %s", ot.TypeName))
				return fmt.Errorf("failed to create object type %s: %w", ot.TypeName, err)
			}
		}
		spinner.Success(fmt.Sprintf("Inserted %d object types", len(r.objectTypes)))
	}

	if len(r.objects) > 0 {
		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Inserting %d objects...", len(r.objects)))
		for _, o := range r.objects {
			if verbose {
				spinner.UpdateText(fmt.Sprintf("Inserting object: %s", *o.Alias))
			}
			if _, err := azc.CreateObject(ctx, o.ID, o.TypeID, *o.Alias); err != nil {
				spinner.Fail(fmt.Sprintf("Failed to insert object: %s", *o.Alias))
				return fmt.Errorf("failed to create object %s: %w", *o.Alias, err)
			}
		}
		spinner.Success(fmt.Sprintf("Inserted %d objects", len(r.objects)))
	}

	if len(r.edgeTypes) > 0 {
		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Inserting %d edge types...", len(r.edgeTypes)))
		for _, et := range r.edgeTypes {
			if verbose {
				spinner.UpdateText(fmt.Sprintf("Inserting edge type: %s", et.TypeName))
			}
			if _, err := azc.CreateEdgeType(ctx, et.ID, et.SourceObjectTypeID, et.TargetObjectTypeID, et.TypeName, et.Attributes); err != nil {
				spinner.Fail(fmt.Sprintf("Failed to insert edge type: %s", et.TypeName))
				return fmt.Errorf("failed to create edge type %s: %w", et.TypeName, err)
			}
		}
		spinner.Success(fmt.Sprintf("Inserted %d edge types", len(r.edgeTypes)))
	}

	if len(r.edges) > 0 {
		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Inserting %d edges...", len(r.edges)))
		for i, e := range r.edges {
			if verbose {
				spinner.UpdateText(fmt.Sprintf("Inserting edge %d/%d", i+1, len(r.edges)))
			}
			if _, err := azc.CreateEdge(ctx, e.ID, e.SourceObjectID, e.TargetObjectID, e.EdgeTypeID); err != nil {
				spinner.Fail("Failed to insert edge")
				return fmt.Errorf("failed to create edge: %w", err)
			}
		}
		spinner.Success(fmt.Sprintf("Inserted %d edges", len(r.edges)))
	}

	return nil
}

func (r *resources) delete(ctx context.Context, azc *authz.Client, verbose bool) error {
	// Delete in reverse dependency order: Edges -> EdgeTypes -> Objects -> ObjectTypes

	if len(r.edges) > 0 {
		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Deleting %d edges...", len(r.edges)))
		for i, e := range r.edges {
			if verbose {
				spinner.UpdateText(fmt.Sprintf("Deleting edge %d/%d", i+1, len(r.edges)))
			}
			if err := azc.DeleteEdge(ctx, e.ID); err != nil {
				spinner.Fail("Failed to delete edge")
				return fmt.Errorf("failed to delete edge: %w", err)
			}
		}
		spinner.Success(fmt.Sprintf("Deleted %d edges", len(r.edges)))
	}

	if len(r.edgeTypes) > 0 {
		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Deleting %d edge types...", len(r.edgeTypes)))
		for _, et := range r.edgeTypes {
			if verbose {
				spinner.UpdateText(fmt.Sprintf("Deleting edge type: %s", et.TypeName))
			}
			if err := azc.DeleteEdgeType(ctx, et.ID); err != nil {
				spinner.Fail(fmt.Sprintf("Failed to delete edge type: %s", et.TypeName))
				return fmt.Errorf("failed to delete edge type %s: %w", et.TypeName, err)
			}
		}
		spinner.Success(fmt.Sprintf("Deleted %d edge types", len(r.edgeTypes)))
	}

	if len(r.objects) > 0 {
		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Deleting %d objects...", len(r.objects)))
		for _, o := range r.objects {
			if verbose {
				spinner.UpdateText(fmt.Sprintf("Deleting object: %s", *o.Alias))
			}
			if err := azc.DeleteObject(ctx, o.ID); err != nil {
				spinner.Fail(fmt.Sprintf("Failed to delete object: %s", *o.Alias))
				return fmt.Errorf("failed to delete object: %w", err)
			}
		}
		spinner.Success(fmt.Sprintf("Deleted %d objects", len(r.objects)))
	}

	if len(r.objectTypes) > 0 {
		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Deleting %d object types...", len(r.objectTypes)))
		for _, ot := range r.objectTypes {
			if verbose {
				spinner.UpdateText(fmt.Sprintf("Deleting object type: %s", ot.TypeName))
			}
			if err := azc.DeleteObjectType(ctx, ot.ID); err != nil {
				spinner.Fail(fmt.Sprintf("Failed to delete object type: %s", ot.TypeName))
				return fmt.Errorf("failed to delete object type %s: %w", ot.TypeName, err)
			}
		}
		spinner.Success(fmt.Sprintf("Deleted %d object types", len(r.objectTypes)))
	}

	return nil
}

func (r *resources) diff(ctx context.Context, src, dst *resources, verbose bool) {
	// Build lookup maps for destination resources
	dstEdgeTypeMap := make(map[uuid.UUID]*authz.EdgeType, len(dst.edgeTypes))
	for i := range dst.edgeTypes {
		dstEdgeTypeMap[dst.edgeTypes[i].ID] = &dst.edgeTypes[i]
	}

	dstEdgeMap := make(map[uuid.UUID]*authz.Edge, len(dst.edges))
	for i := range dst.edges {
		dstEdgeMap[dst.edges[i].ID] = &dst.edges[i]
	}

	dstObjectTypeMap := make(map[uuid.UUID]*authz.ObjectType, len(dst.objectTypes))
	for i := range dst.objectTypes {
		dstObjectTypeMap[dst.objectTypes[i].ID] = &dst.objectTypes[i]
	}

	dstObjectMap := make(map[uuid.UUID]*authz.Object, len(dst.objects))
	for i := range dst.objects {
		dstObjectMap[dst.objects[i].ID] = &dst.objects[i]
	}

	// Find differences for each resource type
	for _, srcEdgeType := range src.edgeTypes {
		if dstEdgeType, exists := dstEdgeTypeMap[srcEdgeType.ID]; !exists || !srcEdgeType.EqualsIgnoringID(dstEdgeType) {
			r.edgeTypes = append(r.edgeTypes, srcEdgeType)
		}
	}

	for _, srcEdge := range src.edges {
		if dstEdge, exists := dstEdgeMap[srcEdge.ID]; !exists || !srcEdge.EqualsIgnoringID(dstEdge) {
			r.edges = append(r.edges, srcEdge)
		}
	}

	for _, srcObjectType := range src.objectTypes {
		if dstObjectType, exists := dstObjectTypeMap[srcObjectType.ID]; !exists || !srcObjectType.EqualsIgnoringID(dstObjectType) {
			r.objectTypes = append(r.objectTypes, srcObjectType)
		}
	}

	for _, srcObject := range src.objects {
		if dstObject, exists := dstObjectMap[srcObject.ID]; !exists || !srcObject.EqualsIgnoringID(dstObject) {
			r.objects = append(r.objects, srcObject)
		}
	}
}

func (r *resources) fetchEdgeTypes(ctx context.Context, azc *authz.Client) error {
	edgeTypes, err := azc.ListEdgeTypes(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	r.edgeTypes = edgeTypes
	return nil
}

func (r *resources) fetchEdges(ctx context.Context, azc *authz.Client) error {
	var edges []authz.Edge
	cursor := pagination.CursorBegin

	for {
		resp, err := azc.ListEdges(ctx, authz.Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			return ucerr.Wrap(err)
		}

		edges = append(edges, resp.Data...)
		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}

	r.edges = edges
	return nil
}

func (r *resources) fetchObjectTypes(ctx context.Context, azc *authz.Client) error {
	objectTypes, err := azc.ListObjectTypes(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	r.objectTypes = objectTypes
	return nil
}

func (r *resources) fetchObjects(ctx context.Context, azc *authz.Client) error {
	var objects []authz.Object
	cursor := pagination.CursorBegin

	for {
		resp, err := azc.ListObjects(ctx, authz.Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			return ucerr.Wrap(err)
		}

		objects = append(objects, resp.Data...)
		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}

	r.objects = objects
	return nil
}