// Copyright 2014, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vtgate

import (
	"encoding/hex"
	"fmt"

	"github.com/youtube/vitess/go/sqltypes"
	"github.com/youtube/vitess/go/vt/sqlannotation"
	"github.com/youtube/vitess/go/vt/topo"
	"github.com/youtube/vitess/go/vt/vtgate/planbuilder"
	"golang.org/x/net/context"

	querypb "github.com/youtube/vitess/go/vt/proto/query"
	topodatapb "github.com/youtube/vitess/go/vt/proto/topodata"
	vtgatepb "github.com/youtube/vitess/go/vt/proto/vtgate"
)

const (
	ksidName = "keyspace_id"
)

// Router is the layer to route queries to the correct shards
// based on the values in the query.
type Router struct {
	serv        topo.SrvTopoServer
	cell        string
	planner     *Planner
	scatterConn *ScatterConn
}

type scatterParams struct {
	query, ks string
	shardVars map[string]map[string]interface{}
}

func newScatterParams(query, ks string, bv map[string]interface{}, shards []string) *scatterParams {
	shardVars := make(map[string]map[string]interface{}, len(shards))
	for _, shard := range shards {
		shardVars[shard] = bv
	}
	return &scatterParams{
		query:     query,
		ks:        ks,
		shardVars: shardVars,
	}
}

// NewRouter creates a new Router.
func NewRouter(serv topo.SrvTopoServer, cell string, vschema *planbuilder.VSchema, statsName string, scatterConn *ScatterConn) *Router {
	return &Router{
		serv:        serv,
		cell:        cell,
		planner:     NewPlanner(vschema, 5000),
		scatterConn: scatterConn,
	}
}

// Execute routes a non-streaming query.
func (rtr *Router) Execute(ctx context.Context, sql string, bindVars map[string]interface{}, tabletType topodatapb.TabletType, session *vtgatepb.Session, notInTransaction bool) (*sqltypes.Result, error) {
	if bindVars == nil {
		bindVars = make(map[string]interface{})
	}
	vcursor := newRequestContext(ctx, sql, bindVars, tabletType, session, notInTransaction, rtr)
	plan, err := rtr.planner.GetPlan(sql)
	if err != nil {
		return nil, err
	}
	return rtr.execInstruction(vcursor, plan.Instructions)
}

func (rtr *Router) execInstruction(vcursor *requestContext, instruction interface{}) (*sqltypes.Result, error) {
	switch instruction := instruction.(type) {
	case *planbuilder.Join:
		return rtr.execJoin(vcursor, instruction)
	case *planbuilder.Route:
		return rtr.execRoute(vcursor, instruction)
	}
	panic("unreachable")
}

func (rtr *Router) execJoin(vcursor *requestContext, join *planbuilder.Join) (*sqltypes.Result, error) {
	lresult, err := rtr.execInstruction(vcursor, join.Left)
	if err != nil {
		return nil, err
	}
	saved := copyBindVars(vcursor.JoinVars)
	defer func() { vcursor.JoinVars = saved }()
	result := &sqltypes.Result{}
	if len(lresult.Rows) == 0 {
		// We still need field info from the RHS.
		// TODO(sougou): Change this to use an impossible query.
		for k := range join.Vars {
			vcursor.JoinVars[k] = nil
		}
		rresult, err := rtr.execInstruction(vcursor, join.Right)
		if err != nil {
			return nil, err
		}
		result.Fields = joinFields(lresult.Fields, rresult.Fields, join.Cols)
		return result, nil
	}
	for _, lrow := range lresult.Rows {
		for k, col := range join.Vars {
			vcursor.JoinVars[k] = lrow[col]
		}
		rresult, err := rtr.execInstruction(vcursor, join.Right)
		if err != nil {
			return nil, err
		}
		result.Fields = joinFields(lresult.Fields, rresult.Fields, join.Cols)
		if join.IsLeft && len(rresult.Rows) == 0 {
			rresult.Rows = [][]sqltypes.Value{make([]sqltypes.Value, len(rresult.Fields))}
		}
		for _, rrow := range rresult.Rows {
			result.Rows = append(result.Rows, joinRows(lrow, rrow, join.Cols))
		}
		result.RowsAffected += uint64(len(rresult.Rows))
	}
	return result, nil
}

func copyBindVars(bindVars map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range bindVars {
		out[k] = v
	}
	return out
}

func joinFields(lfields, rfields []*querypb.Field, cols []int) []*querypb.Field {
	if lfields == nil || rfields == nil {
		return nil
	}
	fields := make([]*querypb.Field, len(cols))
	for i, index := range cols {
		if index < 0 {
			fields[i] = lfields[-index-1]
			continue
		}
		fields[i] = rfields[index-1]
	}
	return fields
}

func joinRows(lrow, rrow []sqltypes.Value, cols []int) []sqltypes.Value {
	row := make([]sqltypes.Value, len(cols))
	for i, index := range cols {
		if index < 0 {
			row[i] = lrow[-index-1]
			continue
		}
		row[i] = rrow[index-1]
	}
	return row
}

func (rtr *Router) execRoute(vcursor *requestContext, route *planbuilder.Route) (*sqltypes.Result, error) {
	switch route.PlanID {
	case planbuilder.UpdateEqual:
		return rtr.execUpdateEqual(vcursor, route)
	case planbuilder.DeleteEqual:
		return rtr.execDeleteEqual(vcursor, route)
	case planbuilder.InsertSharded:
		return rtr.execInsertSharded(vcursor, route)
	}

	var err error
	var params *scatterParams
	switch route.PlanID {
	case planbuilder.SelectUnsharded, planbuilder.UpdateUnsharded,
		planbuilder.DeleteUnsharded, planbuilder.InsertUnsharded:
		params, err = rtr.paramsUnsharded(vcursor, route)
	case planbuilder.SelectEqual, planbuilder.SelectEqualUnique:
		params, err = rtr.paramsSelectEqual(vcursor, route)
	case planbuilder.SelectIN:
		params, err = rtr.paramsSelectIN(vcursor, route)
	case planbuilder.SelectScatter:
		params, err = rtr.paramsSelectScatter(vcursor, route)
	default:
		// TODO(sougou): improve error.
		return nil, fmt.Errorf("unsupported query route: %v", route)
	}
	if err != nil {
		return nil, err
	}
	saved := copyBindVars(vcursor.bindVars)
	defer func() { vcursor.bindVars = saved }()
	for k := range route.JoinVars {
		vcursor.bindVars[k] = vcursor.JoinVars[k]
	}
	return rtr.scatterConn.ExecuteMulti(
		vcursor.ctx,
		params.query,
		params.ks,
		params.shardVars,
		vcursor.tabletType,
		NewSafeSession(vcursor.session),
		vcursor.notInTransaction,
	)
}

// StreamExecute executes a streaming query.
func (rtr *Router) StreamExecute(ctx context.Context, sql string, bindVars map[string]interface{}, tabletType topodatapb.TabletType, sendReply func(*sqltypes.Result) error) error {
	if bindVars == nil {
		bindVars = make(map[string]interface{})
	}
	vcursor := newRequestContext(ctx, sql, bindVars, tabletType, nil, false, rtr)
	plan, err := rtr.planner.GetPlan(sql)
	if err != nil {
		return err
	}
	return rtr.streamExecInstruction(vcursor, plan.Instructions, sendReply)
}

func (rtr *Router) streamExecInstruction(vcursor *requestContext, instruction interface{}, sendReply func(*sqltypes.Result) error) error {
	switch instruction := instruction.(type) {
	case *planbuilder.Join:
		return rtr.streamExecJoin(vcursor, instruction, sendReply)
	case *planbuilder.Route:
		return rtr.streamExecRoute(vcursor, instruction, sendReply)
	}
	panic("unreachable")
}

func (rtr *Router) streamExecJoin(vcursor *requestContext, join *planbuilder.Join, sendReply func(*sqltypes.Result) error) error {
	fieldSent := false
	var rhsLen int
	err := rtr.streamExecInstruction(vcursor, join.Left, func(lresult *sqltypes.Result) error {
		saved := copyBindVars(vcursor.JoinVars)
		defer func() { vcursor.JoinVars = saved }()
		for _, lrow := range lresult.Rows {
			for k, col := range join.Vars {
				vcursor.JoinVars[k] = lrow[col]
			}
			rowSent := false
			err := rtr.streamExecInstruction(vcursor, join.Right, func(rresult *sqltypes.Result) error {
				result := &sqltypes.Result{}
				if !fieldSent {
					result.Fields = joinFields(lresult.Fields, rresult.Fields, join.Cols)
					fieldSent = true
					rhsLen = len(rresult.Fields)
				}
				if len(result.Rows) != 0 {
					rowSent = true
				}
				for _, rrow := range rresult.Rows {
					result.Rows = append(result.Rows, joinRows(lrow, rrow, join.Cols))
				}
				return sendReply(result)
			})
			if err != nil {
				return err
			}
			if join.IsLeft && !rowSent {
				result := &sqltypes.Result{}
				result.Rows = [][]sqltypes.Value{joinRows(
					lrow,
					make([]sqltypes.Value, rhsLen),
					join.Cols,
				)}
				err := sendReply(result)
				if err != nil {
					return err
				}
			}
		}
		if !fieldSent {
			for k := range join.Vars {
				vcursor.JoinVars[k] = nil
			}
			return rtr.streamExecInstruction(vcursor, join.Right, func(rresult *sqltypes.Result) error {
				result := &sqltypes.Result{}
				result.Fields = joinFields(lresult.Fields, rresult.Fields, join.Cols)
				fieldSent = true
				rhsLen = len(rresult.Fields)
				return sendReply(result)
			})
		}
		return nil
	})
	return err
}

func (rtr *Router) streamExecRoute(vcursor *requestContext, route *planbuilder.Route, sendReply func(*sqltypes.Result) error) error {
	var err error
	var params *scatterParams
	switch route.PlanID {
	case planbuilder.SelectUnsharded:
		params, err = rtr.paramsUnsharded(vcursor, route)
	case planbuilder.SelectEqual, planbuilder.SelectEqualUnique:
		params, err = rtr.paramsSelectEqual(vcursor, route)
	case planbuilder.SelectIN:
		params, err = rtr.paramsSelectIN(vcursor, route)
	case planbuilder.SelectScatter:
		params, err = rtr.paramsSelectScatter(vcursor, route)
	default:
		return fmt.Errorf("query %q cannot be used for streaming", route.Query)
	}
	if err != nil {
		return err
	}
	saved := copyBindVars(vcursor.bindVars)
	defer func() { vcursor.bindVars = saved }()
	for k := range route.JoinVars {
		vcursor.bindVars[k] = vcursor.JoinVars[k]
	}
	return rtr.scatterConn.StreamExecuteMulti(
		vcursor.ctx,
		params.query,
		params.ks,
		params.shardVars,
		vcursor.tabletType,
		sendReply,
	)
}

func (rtr *Router) paramsUnsharded(vcursor *requestContext, route *planbuilder.Route) (*scatterParams, error) {
	ks, _, allShards, err := getKeyspaceShards(vcursor.ctx, rtr.serv, rtr.cell, route.Keyspace.Name, vcursor.tabletType)
	if err != nil {
		return nil, fmt.Errorf("paramsUnsharded: %v", err)
	}
	if len(allShards) != 1 {
		return nil, fmt.Errorf("unsharded keyspace %s has multiple shards", ks)
	}
	return newScatterParams(route.Query, ks, vcursor.bindVars, []string{allShards[0].Name}), nil
}

func (rtr *Router) paramsSelectEqual(vcursor *requestContext, route *planbuilder.Route) (*scatterParams, error) {
	keys, err := rtr.resolveKeys([]interface{}{route.Values}, vcursor.bindVars)
	if err != nil {
		return nil, fmt.Errorf("paramsSelectEqual: %v", err)
	}
	ks, routing, err := rtr.resolveShards(vcursor, keys, route)
	if err != nil {
		return nil, fmt.Errorf("paramsSelectEqual: %v", err)
	}
	return newScatterParams(route.Query, ks, vcursor.bindVars, routing.Shards()), nil
}

func (rtr *Router) paramsSelectIN(vcursor *requestContext, route *planbuilder.Route) (*scatterParams, error) {
	keys, err := rtr.resolveKeys(route.Values.([]interface{}), vcursor.bindVars)
	if err != nil {
		return nil, fmt.Errorf("paramsSelectIN: %v", err)
	}
	ks, routing, err := rtr.resolveShards(vcursor, keys, route)
	if err != nil {
		return nil, fmt.Errorf("paramsSelectEqual: %v", err)
	}
	return &scatterParams{
		query:     route.Query,
		ks:        ks,
		shardVars: routing.ShardVars(vcursor.bindVars),
	}, nil
}

func (rtr *Router) paramsSelectScatter(vcursor *requestContext, route *planbuilder.Route) (*scatterParams, error) {
	ks, _, allShards, err := getKeyspaceShards(vcursor.ctx, rtr.serv, rtr.cell, route.Keyspace.Name, vcursor.tabletType)
	if err != nil {
		return nil, fmt.Errorf("paramsSelectScatter: %v", err)
	}
	var shards []string
	for _, shard := range allShards {
		shards = append(shards, shard.Name)
	}
	return newScatterParams(route.Query, ks, vcursor.bindVars, shards), nil
}

func (rtr *Router) execUpdateEqual(vcursor *requestContext, route *planbuilder.Route) (*sqltypes.Result, error) {
	keys, err := rtr.resolveKeys([]interface{}{route.Values}, vcursor.bindVars)
	if err != nil {
		return nil, fmt.Errorf("execUpdateEqual: %v", err)
	}
	ks, shard, ksid, err := rtr.resolveSingleShard(vcursor, keys[0], route)
	if err != nil {
		return nil, fmt.Errorf("execUpdateEqual: %v", err)
	}
	if len(ksid) == 0 {
		return &sqltypes.Result{}, nil
	}
	vcursor.bindVars[ksidName] = string(ksid)
	rewritten := sqlannotation.AddKeyspaceID(route.Query, ksid)
	return rtr.scatterConn.Execute(
		vcursor.ctx,
		rewritten,
		vcursor.bindVars,
		ks,
		[]string{shard},
		vcursor.tabletType,
		NewSafeSession(vcursor.session),
		vcursor.notInTransaction)
}

func (rtr *Router) execDeleteEqual(vcursor *requestContext, route *planbuilder.Route) (*sqltypes.Result, error) {
	keys, err := rtr.resolveKeys([]interface{}{route.Values}, vcursor.bindVars)
	if err != nil {
		return nil, fmt.Errorf("execDeleteEqual: %v", err)
	}
	ks, shard, ksid, err := rtr.resolveSingleShard(vcursor, keys[0], route)
	if err != nil {
		return nil, fmt.Errorf("execDeleteEqual: %v", err)
	}
	if len(ksid) == 0 {
		return &sqltypes.Result{}, nil
	}
	if route.Subquery != "" {
		err = rtr.deleteVindexEntries(vcursor, route, ks, shard, ksid)
		if err != nil {
			return nil, fmt.Errorf("execDeleteEqual: %v", err)
		}
	}
	vcursor.bindVars[ksidName] = string(ksid)
	rewritten := sqlannotation.AddKeyspaceID(route.Query, ksid)
	return rtr.scatterConn.Execute(
		vcursor.ctx,
		rewritten,
		vcursor.bindVars,
		ks,
		[]string{shard},
		vcursor.tabletType,
		NewSafeSession(vcursor.session),
		vcursor.notInTransaction)
}

func (rtr *Router) execInsertSharded(vcursor *requestContext, route *planbuilder.Route) (*sqltypes.Result, error) {
	input := route.Values.([]interface{})
	keys, err := rtr.resolveKeys(input, vcursor.bindVars)
	if err != nil {
		return nil, fmt.Errorf("execInsertSharded: %v", err)
	}
	ksid, generated, err := rtr.handlePrimary(vcursor, keys[0], route.Table.ColVindexes[0], vcursor.bindVars)
	if err != nil {
		return nil, fmt.Errorf("execInsertSharded: %v", err)
	}
	ks, shard, err := rtr.getRouting(vcursor.ctx, route.Keyspace.Name, vcursor.tabletType, ksid)
	if err != nil {
		return nil, fmt.Errorf("execInsertSharded: %v", err)
	}
	for i := 1; i < len(keys); i++ {
		newgen, err := rtr.handleNonPrimary(vcursor, keys[i], route.Table.ColVindexes[i], vcursor.bindVars, ksid)
		if err != nil {
			return nil, err
		}
		if newgen != 0 {
			if generated != 0 {
				return nil, fmt.Errorf("insert generated more than one value")
			}
			generated = newgen
		}
	}
	vcursor.bindVars[ksidName] = string(ksid)
	rewritten := sqlannotation.AddKeyspaceID(route.Query, ksid)
	result, err := rtr.scatterConn.Execute(
		vcursor.ctx,
		rewritten,
		vcursor.bindVars,
		ks,
		[]string{shard},
		vcursor.tabletType,
		NewSafeSession(vcursor.session),
		vcursor.notInTransaction)
	if err != nil {
		return nil, fmt.Errorf("execInsertSharded: %v", err)
	}
	if generated != 0 {
		if result.InsertID != 0 {
			return nil, fmt.Errorf("vindex and db generated a value each for insert")
		}
		result.InsertID = uint64(generated)
	}
	return result, nil
}

func (rtr *Router) resolveKeys(vals []interface{}, bindVars map[string]interface{}) (keys []interface{}, err error) {
	keys = make([]interface{}, 0, len(vals))
	for _, val := range vals {
		if v, ok := val.(string); ok {
			val, ok = bindVars[v[1:]]
			if !ok {
				return nil, fmt.Errorf("could not find bind var %s", v)
			}
		}
		if v, ok := val.([]byte); ok {
			val = string(v)
		}
		keys = append(keys, val)
	}
	return keys, nil
}

func (rtr *Router) resolveShards(vcursor *requestContext, vindexKeys []interface{}, route *planbuilder.Route) (newKeyspace string, routing routingMap, err error) {
	newKeyspace, _, allShards, err := getKeyspaceShards(vcursor.ctx, rtr.serv, rtr.cell, route.Keyspace.Name, vcursor.tabletType)
	if err != nil {
		return "", nil, err
	}
	routing = make(routingMap)
	switch mapper := route.Vindex.(type) {
	case planbuilder.Unique:
		ksids, err := mapper.Map(vcursor, vindexKeys)
		if err != nil {
			return "", nil, err
		}
		for i, ksid := range ksids {
			if len(ksid) == 0 {
				continue
			}
			shard, err := getShardForKeyspaceID(allShards, ksid)
			if err != nil {
				return "", nil, err
			}
			routing.Add(shard, vindexKeys[i])
		}
	case planbuilder.NonUnique:
		ksidss, err := mapper.Map(vcursor, vindexKeys)
		if err != nil {
			return "", nil, err
		}
		for i, ksids := range ksidss {
			for _, ksid := range ksids {
				shard, err := getShardForKeyspaceID(allShards, ksid)
				if err != nil {
					return "", nil, err
				}
				routing.Add(shard, vindexKeys[i])
			}
		}
	default:
		panic("unexpected")
	}
	return newKeyspace, routing, nil
}

func (rtr *Router) resolveSingleShard(vcursor *requestContext, vindexKey interface{}, route *planbuilder.Route) (newKeyspace, shard string, ksid []byte, err error) {
	newKeyspace, _, allShards, err := getKeyspaceShards(vcursor.ctx, rtr.serv, rtr.cell, route.Keyspace.Name, vcursor.tabletType)
	if err != nil {
		return "", "", nil, err
	}
	mapper := route.Vindex.(planbuilder.Unique)
	ksids, err := mapper.Map(vcursor, []interface{}{vindexKey})
	if err != nil {
		return "", "", nil, err
	}
	ksid = ksids[0]
	if len(ksid) == 0 {
		return "", "", ksid, nil
	}
	shard, err = getShardForKeyspaceID(allShards, ksid)
	if err != nil {
		return "", "", nil, err
	}
	return newKeyspace, shard, ksid, nil
}

func (rtr *Router) deleteVindexEntries(vcursor *requestContext, route *planbuilder.Route, ks, shard string, ksid []byte) error {
	result, err := rtr.scatterConn.Execute(
		vcursor.ctx,
		route.Subquery,
		vcursor.bindVars,
		ks,
		[]string{shard},
		vcursor.tabletType,
		NewSafeSession(vcursor.session),
		vcursor.notInTransaction)
	if err != nil {
		return err
	}
	if len(result.Rows) == 0 {
		return nil
	}
	for i, colVindex := range route.Table.Owned {
		keys := make(map[interface{}]bool)
		for _, row := range result.Rows {
			switch k := row[i].ToNative().(type) {
			case []byte:
				keys[string(k)] = true
			default:
				keys[k] = true
			}
		}
		var ids []interface{}
		for k := range keys {
			ids = append(ids, k)
		}
		switch vindex := colVindex.Vindex.(type) {
		case planbuilder.Functional:
			if err = vindex.Delete(vcursor, ids, ksid); err != nil {
				return err
			}
		case planbuilder.Lookup:
			if err = vindex.Delete(vcursor, ids, ksid); err != nil {
				return err
			}
		default:
			panic("unexpceted")
		}
	}
	return nil
}

func (rtr *Router) handlePrimary(vcursor *requestContext, vindexKey interface{}, colVindex *planbuilder.ColVindex, bv map[string]interface{}) (ksid []byte, generated int64, err error) {
	if colVindex.Owned {
		if vindexKey == nil {
			generator, ok := colVindex.Vindex.(planbuilder.FunctionalGenerator)
			if !ok {
				return nil, 0, fmt.Errorf("value must be supplied for column %s", colVindex.Col)
			}
			generated, err = generator.Generate(vcursor)
			vindexKey = generated
			if err != nil {
				return nil, 0, err
			}
		} else {
			// TODO(sougou): I think we have to ignore dup key error if this was
			// an upsert. For now, I'm punting on this because this would be a very
			// uncommon use case. We should revisit this when work on v3 resumes.
			err = colVindex.Vindex.(planbuilder.Functional).Create(vcursor, vindexKey)
			if err != nil {
				return nil, 0, err
			}
		}
	}
	if vindexKey == nil {
		return nil, 0, fmt.Errorf("value must be supplied for column %s", colVindex.Col)
	}
	mapper := colVindex.Vindex.(planbuilder.Unique)
	ksids, err := mapper.Map(vcursor, []interface{}{vindexKey})
	if err != nil {
		return nil, 0, err
	}
	ksid = ksids[0]
	if len(ksid) == 0 {
		return nil, 0, fmt.Errorf("could not map %v to a keyspace id", vindexKey)
	}
	bv["_"+colVindex.Col] = vindexKey
	return ksid, generated, nil
}

func (rtr *Router) handleNonPrimary(vcursor *requestContext, vindexKey interface{}, colVindex *planbuilder.ColVindex, bv map[string]interface{}, ksid []byte) (generated int64, err error) {
	if colVindex.Owned {
		if vindexKey == nil {
			generator, ok := colVindex.Vindex.(planbuilder.LookupGenerator)
			if !ok {
				return 0, fmt.Errorf("value must be supplied for column %s", colVindex.Col)
			}
			generated, err = generator.Generate(vcursor, ksid)
			vindexKey = generated
			if err != nil {
				return 0, err
			}
		} else {
			err = colVindex.Vindex.(planbuilder.Lookup).Create(vcursor, vindexKey, ksid)
			if err != nil {
				return 0, err
			}
		}
	} else {
		if vindexKey == nil {
			reversible, ok := colVindex.Vindex.(planbuilder.Reversible)
			if !ok {
				return 0, fmt.Errorf("value must be supplied for column %s", colVindex.Col)
			}
			vindexKey, err = reversible.ReverseMap(vcursor, ksid)
			if err != nil {
				return 0, err
			}
			if vindexKey == nil {
				return 0, fmt.Errorf("could not compute value for column %v", colVindex.Col)
			}
		} else {
			ok, err := colVindex.Vindex.Verify(vcursor, vindexKey, ksid)
			if err != nil {
				return 0, err
			}
			if !ok {
				return 0, fmt.Errorf("value %v for column %s does not map to keyspace id %v", vindexKey, colVindex.Col, hex.EncodeToString(ksid))
			}
		}
	}
	bv["_"+colVindex.Col] = vindexKey
	return generated, nil
}

func (rtr *Router) getRouting(ctx context.Context, keyspace string, tabletType topodatapb.TabletType, ksid []byte) (newKeyspace, shard string, err error) {
	newKeyspace, _, allShards, err := getKeyspaceShards(ctx, rtr.serv, rtr.cell, keyspace, tabletType)
	if err != nil {
		return "", "", err
	}
	shard, err = getShardForKeyspaceID(allShards, ksid)
	if err != nil {
		return "", "", err
	}
	return newKeyspace, shard, nil
}
