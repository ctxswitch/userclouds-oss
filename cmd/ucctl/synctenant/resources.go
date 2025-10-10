package synctenant

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

type Resources struct {
	edgeTypes   []authz.EdgeType
	edges       []authz.Edge
	objectTypes []authz.ObjectType
	objects     []authz.Object
}

func NewResources() *Resources {
	return &Resources{
		edgeTypes:   make([]authz.EdgeType, 0),
		edges:       make([]authz.Edge, 0),
		objectTypes: make([]authz.ObjectType, 0),
		objects:     make([]authz.Object, 0),
	}
}

func (r *Resources) Get(ctx context.Context, azc *authz.Client) error {
	uclog.Infof(ctx, "Fetching ObjectTypes")
	if err := r.readAllObjectTypes(ctx, azc); err != nil {
		return err
	}
	uclog.Infof(ctx, "Fetched %d object types", len(r.objectTypes))

	uclog.Infof(ctx, "Fetching objects")
	if err := r.readAllObjects(ctx, azc); err != nil {
		return err
	}
	uclog.Infof(ctx, "Fetched %d objects", len(r.objects))

	uclog.Infof(ctx, "Fetching edgeTypes")
	if err := r.readAllEdgeTypes(ctx, azc); err != nil {
		return err
	}
	uclog.Infof(ctx, "Fetched %d edgeTypes", len(r.edgeTypes))

	uclog.Infof(ctx, "Fetching edges")
	if err := r.readAllEdges(ctx, azc); err != nil {
		return err
	}
	uclog.Infof(ctx, "Fetched %d edges", len(r.edges))

	return nil
}

func (r *Resources) Insert(ctx context.Context, azc *authz.Client) error {
	uclog.Infof(ctx, "Inserting ObjectTypes")
	for _, ot := range r.objectTypes {
		_, err := azc.CreateObjectType(ctx, ot.ID, ot.TypeName)
		if err != nil {
			return err
		}
	}
	uclog.Infof(ctx, "Inserted %d ObjectTypes", len(r.objects))

	uclog.Infof(ctx, "Inserting Objects")
	for _, o := range r.objects {
		_, err := azc.CreateObject(ctx, o.ID, o.TypeID, *o.Alias)
		if err != nil {
			return err
		}
	}
	uclog.Infof(ctx, "Inserted %d Objects", len(r.objects))

	uclog.Infof(ctx, "Inserting EdgeTypes")
	for _, et := range r.edgeTypes {
		_, err := azc.CreateEdgeType(ctx, et.ID, et.SourceObjectTypeID, et.TargetObjectTypeID, et.TypeName, et.Attributes)
		if err != nil {
			return err
		}
	}
	uclog.Infof(ctx, "Inserted %d EdgeTypes", len(r.edgeTypes))

	uclog.Infof(ctx, "Inserting Edges")
	for _, e := range r.edges {
		_, err := azc.CreateEdge(ctx, e.ID, e.SourceObjectID, e.TargetObjectID, e.EdgeTypeID)
		if err != nil {
			return err
		}
	}
	uclog.Infof(ctx, "Inserted %d Edges", len(r.edges))

	return nil
}

func (r *Resources) Delete(ctx context.Context, azc *authz.Client) error {
	uclog.Infof(ctx, "Deleting Edges")
	for _, e := range r.edges {
		err := azc.DeleteEdge(ctx, e.ID)
		if err != nil {
			return err
		}
	}
	uclog.Infof(ctx, "Deleted %d Edges", len(r.edges))

	uclog.Infof(ctx, "Deleting EdgeTypes")
	for _, et := range r.edgeTypes {
		err := azc.DeleteEdgeType(ctx, et.ID)
		if err != nil {
			return err
		}
	}
	uclog.Infof(ctx, "Deleted %d EdgeTypes", len(r.edgeTypes))

	uclog.Infof(ctx, "Deleting Objects")
	for _, o := range r.objects {
		err := azc.DeleteObject(ctx, o.ID)
		if err != nil {
			return err
		}
	}
	uclog.Infof(ctx, "Deleted %d Objects", len(r.objects))

	uclog.Infof(ctx, "Deleting ObjectTypes")
	for _, ot := range r.objectTypes {
		err := azc.DeleteObject(ctx, ot.ID)
		if err != nil {
			return err
		}
	}
	uclog.Infof(ctx, "Deleted %d ObjectTypes", len(r.objectTypes))

	return nil
}

func (r *Resources) Diff(ctx context.Context, src *Resources, dst *Resources) {
	dstEdgeTypeMap := make(map[uuid.UUID]*authz.EdgeType)
	for i := range dst.edgeTypes {
		dstEdgeTypeMap[dst.edgeTypes[i].ID] = &dst.edgeTypes[i]
	}

	dstEdgeMap := make(map[uuid.UUID]*authz.Edge)
	for i := range dst.edges {
		dstEdgeMap[dst.edges[i].ID] = &dst.edges[i]
	}

	dstObjectTypeMap := make(map[uuid.UUID]*authz.ObjectType)
	for i := range dst.objectTypes {
		dstObjectTypeMap[dst.objectTypes[i].ID] = &dst.objectTypes[i]
	}

	dstObjectMap := make(map[uuid.UUID]*authz.Object)
	for i := range dst.objects {
		dstObjectMap[dst.objects[i].ID] = &dst.objects[i]
	}

	for _, srcEdgeType := range src.edgeTypes {
		if dstEdgeType, exists := dstEdgeTypeMap[srcEdgeType.ID]; !exists || !srcEdgeType.EqualsIgnoringID(dstEdgeType) {
			r.edgeTypes = append(r.edgeTypes, srcEdgeType)
		}
	}
	uclog.Infof(ctx, "Diff: %d EdgeTypes", len(r.edgeTypes))

	for _, srcEdge := range src.edges {
		if dstEdge, exists := dstEdgeMap[srcEdge.ID]; !exists || !srcEdge.EqualsIgnoringID(dstEdge) {
			r.edges = append(r.edges, srcEdge)
		}
	}
	uclog.Infof(ctx, "Diff: %d Edges", len(r.edges))

	for _, srcObjectType := range src.objectTypes {
		if dstObjectType, exists := dstObjectTypeMap[srcObjectType.ID]; !exists || !srcObjectType.EqualsIgnoringID(dstObjectType) {
			r.objectTypes = append(r.objectTypes, srcObjectType)
		}
	}
	uclog.Infof(ctx, "Diff: %d ObjectTypes", len(r.objectTypes))

	for _, srcObject := range src.objects {
		if dstObject, exists := dstObjectMap[srcObject.ID]; !exists || !srcObject.EqualsIgnoringID(dstObject) {
			r.objects = append(r.objects, srcObject)
		}
	}
	uclog.Infof(ctx, "Diff: %d Objects", len(r.objects))
}

func (r *Resources) readAllEdgeTypes(ctx context.Context, azc *authz.Client) error {
	edgeTypes, err := azc.ListEdgeTypes(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	r.edgeTypes = edgeTypes
	return nil
}

func (r *Resources) readAllEdges(ctx context.Context, azc *authz.Client) error {
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

func (r *Resources) readAllObjectTypes(ctx context.Context, azc *authz.Client) error {
	objectTypes, err := azc.ListObjectTypes(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	r.objectTypes = objectTypes
	return nil
}

func (r *Resources) readAllObjects(ctx context.Context, azc *authz.Client) error {
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
