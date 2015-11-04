// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	mproto "github.com/youtube/vitess/go/mysql/proto"
	"github.com/youtube/vitess/go/vt/key"
	tproto "github.com/youtube/vitess/go/vt/tabletserver/proto"
	"github.com/youtube/vitess/go/vt/topo"

	pb "github.com/youtube/vitess/go/vt/proto/vtgate"
)

// Query represents a keyspace agnostic query request.
type Query struct {
	CallerID         *tproto.CallerID // only used by BSON
	Sql              string
	BindVariables    map[string]interface{}
	TabletType       topo.TabletType
	Session          *pb.Session
	NotInTransaction bool
}

// QueryShard represents a query request for the
// specified list of shards.
type QueryShard struct {
	CallerID         *tproto.CallerID // only used by BSON
	Sql              string
	BindVariables    map[string]interface{}
	Keyspace         string
	Shards           []string
	TabletType       topo.TabletType
	Session          *pb.Session
	NotInTransaction bool
}

// KeyspaceIdQuery represents a query request for the
// specified list of keyspace IDs.
type KeyspaceIdQuery struct {
	CallerID         *tproto.CallerID // only used by BSON
	Sql              string
	BindVariables    map[string]interface{}
	Keyspace         string
	KeyspaceIds      []key.KeyspaceId
	TabletType       topo.TabletType
	Session          *pb.Session
	NotInTransaction bool
}

// KeyRangeQuery represents a query request for the
// specified list of keyranges.
type KeyRangeQuery struct {
	CallerID         *tproto.CallerID // only used by BSON
	Sql              string
	BindVariables    map[string]interface{}
	Keyspace         string
	KeyRanges        []key.KeyRange
	TabletType       topo.TabletType
	Session          *pb.Session
	NotInTransaction bool
}

// EntityId represents a tuple of external_id and keyspace_id
type EntityId struct {
	ExternalID interface{}
	KeyspaceID key.KeyspaceId
}

// EntityIdsQuery represents a query request for the specified KeyspaceId map.
type EntityIdsQuery struct {
	CallerID          *tproto.CallerID // only used by BSON
	Sql               string
	BindVariables     map[string]interface{}
	Keyspace          string
	EntityColumnName  string
	EntityKeyspaceIDs []EntityId
	TabletType        topo.TabletType
	Session           *pb.Session
	NotInTransaction  bool
}

// QueryResult is mproto.QueryResult+Session (for now).
type QueryResult struct {
	Result  *mproto.QueryResult
	Session *pb.Session
	// Error field is deprecated, as it only returns a string. New users should use the
	// Err field below, which contains a string and an error code.
	Error string
	Err   *mproto.RPCError
}

// BoundShardQuery represents a single query request for the
// specified list of shards. This is used in a list for BatchQueryShard.
type BoundShardQuery struct {
	Sql           string
	BindVariables map[string]interface{}
	Keyspace      string
	Shards        []string
}

// BatchQueryShard represents a batch query request
// for the specified shards.
type BatchQueryShard struct {
	CallerID      *tproto.CallerID // only used by BSON
	Queries       []BoundShardQuery
	TabletType    topo.TabletType
	AsTransaction bool
	Session       *pb.Session
}

// BoundKeyspaceIdQuery represents a single query request for the
// specified list of keyspace ids. This is used in a list for KeyspaceIdBatchQuery.
type BoundKeyspaceIdQuery struct {
	Sql           string
	BindVariables map[string]interface{}
	Keyspace      string
	KeyspaceIds   []key.KeyspaceId
}

// KeyspaceIdBatchQuery represents a batch query request
// for the specified keyspace IDs.
type KeyspaceIdBatchQuery struct {
	CallerID      *tproto.CallerID // only used by BSON
	Queries       []BoundKeyspaceIdQuery
	TabletType    topo.TabletType
	AsTransaction bool
	Session       *pb.Session
}

// QueryResultList is mproto.QueryResultList+Session
type QueryResultList struct {
	List    []mproto.QueryResult
	Session *pb.Session
	// Error field is deprecated, as it only returns a string. New users should use the
	// Err field below, which contains a string and an error code.
	Error string
	Err   *mproto.RPCError
}

// SplitQueryRequest is a request to split a query into multiple parts
type SplitQueryRequest struct {
	CallerID    *tproto.CallerID // only used by BSON
	Keyspace    string
	Query       tproto.BoundQuery
	SplitColumn string
	SplitCount  int
}

// BeginRequest is the BSON implementation of the proto3 query.BeginRequest
type BeginRequest struct {
	CallerID *tproto.CallerID // only used by BSON
}

// BeginResponse is the BSON implementation of the proto3 vtgate.BeginResponse
type BeginResponse struct {
	// Err is named 'Err' instead of 'Error' (as the proto3 version is) to remain
	// consistent with other BSON structs.
	Err     *mproto.RPCError
	Session *pb.Session
}

// CommitRequest is the BSON implementation of the proto3 vtgate.CommitRequest
type CommitRequest struct {
	CallerID *tproto.CallerID // only used by BSON
	Session  *pb.Session
}

// CommitResponse is the BSON implementation of the proto3 vtgate.CommitResponse
type CommitResponse struct {
	// Err is named 'Err' instead of 'Error' (as the proto3 version is) to remain
	// consistent with other BSON structs.
	Err *mproto.RPCError
}

// RollbackRequest is the BSON implementation of the proto3 vtgate.RollbackRequest
type RollbackRequest struct {
	CallerID *tproto.CallerID // only used by BSON
	Session  *pb.Session
}

// RollbackResponse is the BSON implementation of the proto3 vtgate.RollbackResponse
type RollbackResponse struct {
	// Err is named 'Err' instead of 'Error' (as the proto3 version is) to remain
	// consistent with other BSON structs.
	Err *mproto.RPCError
}
