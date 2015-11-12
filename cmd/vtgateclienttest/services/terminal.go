// Copyright 2015 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package services

import (
	"errors"
	"fmt"

	log "github.com/golang/glog"

	"github.com/youtube/vitess/go/tb"
	"github.com/youtube/vitess/go/vt/vtgate/proto"
	"golang.org/x/net/context"

	topodatapb "github.com/youtube/vitess/go/vt/proto/topodata"
	pbg "github.com/youtube/vitess/go/vt/proto/vtgate"
)

var errTerminal = errors.New("vtgate test client, errTerminal")

// terminalClient implements vtgateservice.VTGateService.
// It is the last client in the chain, and returns a well known error.
type terminalClient struct{}

func newTerminalClient() *terminalClient {
	return &terminalClient{}
}

func (c *terminalClient) Execute(ctx context.Context, sql string, bindVariables map[string]interface{}, tabletType topodatapb.TabletType, session *pbg.Session, notInTransaction bool, reply *proto.QueryResult) error {
	if sql == "quit://" {
		log.Fatal("Received quit:// query. Going down.")
	}
	return errTerminal
}

func (c *terminalClient) ExecuteShards(ctx context.Context, sql string, bindVariables map[string]interface{}, keyspace string, shards []string, tabletType topodatapb.TabletType, session *pbg.Session, notInTransaction bool, reply *proto.QueryResult) error {
	return errTerminal
}

func (c *terminalClient) ExecuteKeyspaceIds(ctx context.Context, sql string, bindVariables map[string]interface{}, keyspace string, keyspaceIds [][]byte, tabletType topodatapb.TabletType, session *pbg.Session, notInTransaction bool, reply *proto.QueryResult) error {
	return errTerminal
}

func (c *terminalClient) ExecuteKeyRanges(ctx context.Context, sql string, bindVariables map[string]interface{}, keyspace string, keyRanges []*topodatapb.KeyRange, tabletType topodatapb.TabletType, session *pbg.Session, notInTransaction bool, reply *proto.QueryResult) error {
	return errTerminal
}

func (c *terminalClient) ExecuteEntityIds(ctx context.Context, sql string, bindVariables map[string]interface{}, keyspace string, entityColumnName string, entityKeyspaceIDs []*pbg.ExecuteEntityIdsRequest_EntityId, tabletType topodatapb.TabletType, session *pbg.Session, notInTransaction bool, reply *proto.QueryResult) error {
	return errTerminal
}

func (c *terminalClient) ExecuteBatchShards(ctx context.Context, queries []proto.BoundShardQuery, tabletType topodatapb.TabletType, asTransaction bool, session *pbg.Session, reply *proto.QueryResultList) error {
	return errTerminal
}

func (c *terminalClient) ExecuteBatchKeyspaceIds(ctx context.Context, queries []proto.BoundKeyspaceIdQuery, tabletType topodatapb.TabletType, asTransaction bool, session *pbg.Session, reply *proto.QueryResultList) error {
	return errTerminal
}

func (c *terminalClient) StreamExecute(ctx context.Context, sql string, bindVariables map[string]interface{}, tabletType topodatapb.TabletType, sendReply func(*proto.QueryResult) error) error {
	return errTerminal
}

func (c *terminalClient) StreamExecuteShards(ctx context.Context, sql string, bindVariables map[string]interface{}, keyspace string, shards []string, tabletType topodatapb.TabletType, sendReply func(*proto.QueryResult) error) error {
	return errTerminal
}

func (c *terminalClient) StreamExecuteKeyspaceIds(ctx context.Context, sql string, bindVariables map[string]interface{}, keyspace string, keyspaceIds [][]byte, tabletType topodatapb.TabletType, sendReply func(*proto.QueryResult) error) error {
	return errTerminal
}

func (c *terminalClient) StreamExecuteKeyRanges(ctx context.Context, sql string, bindVariables map[string]interface{}, keyspace string, keyRanges []*topodatapb.KeyRange, tabletType topodatapb.TabletType, sendReply func(*proto.QueryResult) error) error {
	return errTerminal
}

func (c *terminalClient) Begin(ctx context.Context, outSession *pbg.Session) error {
	return errTerminal
}

func (c *terminalClient) Commit(ctx context.Context, inSession *pbg.Session) error {
	return errTerminal
}

func (c *terminalClient) Rollback(ctx context.Context, inSession *pbg.Session) error {
	return errTerminal
}

func (c *terminalClient) SplitQuery(ctx context.Context, keyspace string, sql string, bindVariables map[string]interface{}, splitColumn string, splitCount int) ([]*pbg.SplitQueryResponse_Part, error) {
	return nil, errTerminal
}

func (c *terminalClient) GetSrvKeyspace(ctx context.Context, keyspace string) (*topodatapb.SrvKeyspace, error) {
	return nil, errTerminal
}

func (c *terminalClient) GetSrvShard(ctx context.Context, keyspace, shard string) (*topodatapb.SrvShard, error) {
	return nil, errTerminal
}

func (c *terminalClient) HandlePanic(err *error) {
	if x := recover(); x != nil {
		log.Errorf("Uncaught panic:\n%v\n%s", x, tb.Stack(4))
		*err = fmt.Errorf("uncaught panic: %v", x)
	}
}
