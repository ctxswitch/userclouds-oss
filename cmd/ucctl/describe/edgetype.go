package describe

import (
	stdcontext "context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/pagination"
)

// EdgeTypeCommand handles the describe edgetype subcommand
type EdgeTypeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AutonType       string

	credentials *common.Credentials
}

// RunE executes the describe edgetype command
func (c *EdgeTypeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("edgetype ID or name is required")
	}

	ctx := cmd.Context()
	edgeTypeIdentifier := args[0]

	// Load credentials
	creds, err := common.LoadCredentialsFromContext(
		c.URL,
		c.ClientID,
		c.ClientSecret,
		c.ClientSecretVar,
		"",
	)
	if err != nil {
		return err
	}
	c.credentials = creds

	credOpt, err := c.credentials.GetClientCredentials()
	if err != nil {
		return fmt.Errorf("failed to create client credentials: %w", err)
	}

	// Create AuthZ client
	authzClient, err := authz.NewClient(c.credentials.URL, authz.JSONClient(credOpt))
	if err != nil {
		return fmt.Errorf("failed to create authz client: %w", err)
	}

	// Try to parse as UUID first
	var edgeType *authz.EdgeType
	if edgeTypeID, err := uuid.FromString(edgeTypeIdentifier); err == nil {
		edgeType, err = authzClient.GetEdgeType(ctx, edgeTypeID)
		if err != nil {
			return fmt.Errorf("failed to get edge type: %w", err)
		}
	} else {
		// Search by name
		edgeType, err = c.findEdgeTypeByName(ctx, authzClient, edgeTypeIdentifier)
		if err != nil {
			return err
		}
	}

	// Get source and target object types
	sourceObjType, err := authzClient.GetObjectType(ctx, edgeType.SourceObjectTypeID)
	if err != nil {
		return fmt.Errorf("failed to get source object type: %w", err)
	}

	targetObjType, err := authzClient.GetObjectType(ctx, edgeType.TargetObjectTypeID)
	if err != nil {
		return fmt.Errorf("failed to get target object type: %w", err)
	}

	// Print formatted output
	c.printEdgeTypeDetails(ctx, edgeType, sourceObjType, targetObjType, authzClient)

	return nil
}

func (c *EdgeTypeCommand) findEdgeTypeByName(ctx stdcontext.Context, client *authz.Client, name string) (*authz.EdgeType, error) {
	edgeTypes, err := client.ListEdgeTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list edge types: %w", err)
	}

	for _, et := range edgeTypes {
		if et.TypeName == name {
			return &et, nil
		}
	}

	return nil, fmt.Errorf("edge type not found: %s", name)
}

func (c *EdgeTypeCommand) printEdgeTypeDetails(ctx stdcontext.Context, edgeType *authz.EdgeType, sourceObjType *authz.ObjectType, targetObjType *authz.ObjectType, client *authz.Client) {
	// Header
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Edge Type: %s\n", edgeType.TypeName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Basic Information
	fmt.Println("BASIC INFORMATION")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("  ID:              %s\n", edgeType.ID)
	fmt.Printf("  Type Name:       %s\n", edgeType.TypeName)
	fmt.Printf("  Organization ID: %s\n", edgeType.OrganizationID)
	fmt.Println()

	// Object Type Relationship
	fmt.Println("OBJECT TYPE RELATIONSHIP")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("  Source Type:     %s (%s)\n", sourceObjType.TypeName, sourceObjType.ID)
	fmt.Printf("  Target Type:     %s (%s)\n", targetObjType.TypeName, targetObjType.ID)
	fmt.Println()
	fmt.Printf("  Relationship:    %s → %s\n", sourceObjType.TypeName, targetObjType.TypeName)
	fmt.Println()

	// Attributes (permissions)
	fmt.Println("ATTRIBUTES (PERMISSIONS)")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	if len(edgeType.Attributes) == 0 {
		fmt.Println("  No attributes defined")
	} else {
		for _, attr := range edgeType.Attributes {
			fmt.Printf("  • %s\n", attr.Name)
			permissions := []string{}
			if attr.Direct {
				permissions = append(permissions, "Direct")
			}
			if attr.Inherit {
				permissions = append(permissions, "Inherit")
			}
			if attr.Propagate {
				permissions = append(permissions, "Propagate")
			}
			if len(permissions) > 0 {
				fmt.Printf("    Permissions: %s\n", strings.Join(permissions, ", "))
			}
		}
	}
	fmt.Println()

	// Edges using this type
	fmt.Println("EDGES USING THIS TYPE")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")
	edges := c.getEdgesByType(ctx, client, edgeType.ID)
	if len(edges) == 0 {
		fmt.Println("  No edges found")
	} else {
		for _, edge := range edges {
			sourceInfo := c.getObjectInfo(ctx, client, edge.SourceObjectID)
			targetInfo := c.getObjectInfo(ctx, client, edge.TargetObjectID)
			fmt.Printf("  • %s (%s) → %s (%s)\n",
				edge.SourceObjectID.String()[:8], sourceInfo,
				edge.TargetObjectID.String()[:8], targetInfo)
		}
	}
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func (c *EdgeTypeCommand) getEdgesByType(ctx stdcontext.Context, client *authz.Client, edgeTypeID uuid.UUID) []authz.Edge {
	edges := []authz.Edge{}
	cursor := pagination.CursorBegin

	for {
		filterStr := fmt.Sprintf("('edge_type_id',EQ,'%s')", edgeTypeID)
		paginationOpts := []pagination.Option{pagination.StartingAfter(cursor), pagination.Filter(filterStr)}
		resp, err := client.ListEdges(ctx, authz.Pagination(paginationOpts...))

		if err != nil {
			return edges
		}

		edges = append(edges, resp.Data...)
		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}

	return edges
}

func (c *EdgeTypeCommand) getObjectInfo(ctx stdcontext.Context, client *authz.Client, objectID uuid.UUID) string {
	obj, err := client.GetObject(ctx, objectID)
	if err != nil {
		return "unknown"
	}

	objType, err := client.GetObjectType(ctx, obj.TypeID)
	if err != nil {
		return obj.TypeID.String()[:8]
	}

	info := objType.TypeName
	if obj.Alias != nil && *obj.Alias != "" {
		info += fmt.Sprintf(":%s", *obj.Alias)
	}
	return info
}
