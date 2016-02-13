// Copyright 2016, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package planbuilder

import (
	"errors"

	"github.com/youtube/vitess/go/vt/sqlparser"
)

// processTableExprs analyzes the FROM clause. It produces a planBuilder
// and the associated symtab with all the routes identified.
func processTableExprs(tableExprs sqlparser.TableExprs, vschema *VSchema) (planBuilder, error) {
	if len(tableExprs) != 1 {
		// TODO(sougou): better error message.
		return nil, errors.New("lists are not supported")
	}
	return processTableExpr(tableExprs[0], vschema)
}

// processTableExpr produces a planBuilder subtree and symtab
// for the given TableExpr.
func processTableExpr(tableExpr sqlparser.TableExpr, vschema *VSchema) (planBuilder, error) {
	switch tableExpr := tableExpr.(type) {
	case *sqlparser.AliasedTableExpr:
		return processAliasedTable(tableExpr, vschema)
	case *sqlparser.ParenTableExpr:
		plan, err := processTableExprs(tableExpr.Exprs, vschema)
		// We want to point to the higher level parenthesis because
		// more routes can be merged with this one. If so, the order
		// should be maintained as dictated by the parenthesis.
		if route, ok := plan.(*routeBuilder); ok {
			route.Select.From = sqlparser.TableExprs{tableExpr}
		}
		return plan, err
	case *sqlparser.JoinTableExpr:
		return processJoin(tableExpr, vschema)
	}
	panic("unreachable")
}

// processAliasedTable produces a planBuilder subtree and symtab
// for the given AliasedTableExpr.
func processAliasedTable(tableExpr *sqlparser.AliasedTableExpr, vschema *VSchema) (planBuilder, error) {
	switch expr := tableExpr.Expr.(type) {
	case *sqlparser.TableName:
		route, table, err := getTablePlan(expr, vschema)
		if err != nil {
			return nil, err
		}
		rtb := &routeBuilder{
			Select: sqlparser.Select{From: sqlparser.TableExprs([]sqlparser.TableExpr{tableExpr})},
			symtab: newSymtab(vschema),
			order:  1,
			Route:  route,
		}
		alias := expr.Name
		if tableExpr.As != "" {
			alias = tableExpr.As
		}
		_ = rtb.symtab.AddAlias(alias, table, rtb)
		return rtb, nil
	case *sqlparser.Subquery:
		// TODO(sougou): implement.
		return nil, errors.New("no subqueries")
	}
	panic("unreachable")
}

// getTablePlan produces the initial Route for the specified TableName.
// It also returns the associated vschema info (*Table) so that
// it can be used to create the symbol table entry.
func getTablePlan(tableName *sqlparser.TableName, vschema *VSchema) (*Route, *Table, error) {
	if tableName.Qualifier != "" {
		// TODO(sougou): better error message.
		return nil, nil, errors.New("tablename qualifier not allowed")
	}
	table, err := vschema.FindTable(string(tableName.Name))
	if err != nil {
		return nil, nil, err
	}
	if table.Keyspace.Sharded {
		return &Route{
			PlanID:   SelectScatter,
			Keyspace: table.Keyspace,
			JoinVars: make(map[string]struct{}),
		}, table, nil
	}
	return &Route{
		PlanID:   SelectUnsharded,
		Keyspace: table.Keyspace,
		JoinVars: make(map[string]struct{}),
	}, table, nil
}

// processJoin produces a planBuilder subtree and symtab
// for the given Join. If the left and right nodes can be part
// of the same route, then it's a routeBuilder. Otherwise,
// it's a joinBuilder.
func processJoin(join *sqlparser.JoinTableExpr, vschema *VSchema) (planBuilder, error) {
	switch join.Join {
	case sqlparser.JoinStr, sqlparser.StraightJoinStr, sqlparser.LeftJoinStr:
	default:
		// TODO(sougou): better error message.
		return nil, errors.New("unsupported join")
	}
	lplan, err := processTableExpr(join.LeftExpr, vschema)
	if err != nil {
		return nil, err
	}
	rplan, err := processTableExpr(join.RightExpr, vschema)
	if err != nil {
		return nil, err
	}
	switch lplan := lplan.(type) {
	case *joinBuilder:
		return makejoinBuilder(lplan, rplan, join)
	case *routeBuilder:
		switch rplan := rplan.(type) {
		case *joinBuilder:
			return makejoinBuilder(lplan, rplan, join)
		case *routeBuilder:
			return joinRoutes(lplan, rplan, join)
		}
	}
	panic("unreachable")
}

// makejoinBuilder creates a new joinBuilder node out of the two builders.
// This function is called when the two builders cannot be part of
// the same route.
func makejoinBuilder(lplan, rplan planBuilder, join *sqlparser.JoinTableExpr) (planBuilder, error) {
	// This function converts ON clauses to WHERE clauses. The WHERE clause
	// scope can see all tables, whereas the ON clause can only see the
	// participants of the JOIN. However, since the ON clause doesn't allow
	// external references, and the FROM clause doesn't allow duplicates,
	// it's safe to perform this conversion and still expect the same behavior.

	err := lplan.Symtab().Add(rplan.Symtab())
	if err != nil {
		return nil, err
	}
	setSymtab(rplan, lplan.Symtab())
	assignOrder(rplan, lplan.Order())
	isLeft := false
	if join.Join == sqlparser.LeftJoinStr {
		isLeft = true
	}
	jb := &joinBuilder{
		LeftOrder:  lplan.Order(),
		RightOrder: rplan.Order(),
		Left:       lplan,
		Right:      rplan,
		symtab:     lplan.Symtab(),
		Join: &Join{
			IsLeft: isLeft,
			Left:   getUnderlyingPlan(lplan),
			Right:  getUnderlyingPlan(rplan),
			Vars:   make(map[string]int),
		},
	}
	if isLeft {
		err := processBoolExpr(join.On, rplan, sqlparser.WhereStr)
		if err != nil {
			return nil, err
		}
		setRHS(rplan)
		return jb, nil
	}
	err = processBoolExpr(join.On, jb, sqlparser.WhereStr)
	if err != nil {
		return nil, err
	}
	return jb, nil
}

func setSymtab(plan planBuilder, symtab *symtab) {
	switch plan := plan.(type) {
	case *joinBuilder:
		plan.symtab = symtab
		setSymtab(plan.Left, symtab)
		setSymtab(plan.Right, symtab)
	case *routeBuilder:
		plan.symtab = symtab
	}
}

func getUnderlyingPlan(plan planBuilder) interface{} {
	switch plan := plan.(type) {
	case *joinBuilder:
		return plan.Join
	case *routeBuilder:
		return plan.Route
	}
	panic("unreachable")
}

// assignOrder sets the order for the nodes of the tree based on the
// starting order.
func assignOrder(plan planBuilder, order int) {
	switch plan := plan.(type) {
	case *joinBuilder:
		assignOrder(plan.Left, order)
		plan.LeftOrder = plan.Left.Order()
		assignOrder(plan.Right, plan.Left.Order())
		plan.RightOrder = plan.Right.Order()
	case *routeBuilder:
		plan.order = order + 1
	}
}

// setRHS marks all routes under the plan as RHS of a left join.
func setRHS(plan planBuilder) {
	switch plan := plan.(type) {
	case *joinBuilder:
		setRHS(plan.Left)
		setRHS(plan.Right)
	case *routeBuilder:
		plan.IsRHS = true
	}
}

// joinRoutes attempts to join two routeBuilder objects into one.
// If it's possible, it produces a joined routeBuilder.
// Otherwise, it's a joinBuilder.
func joinRoutes(lRoute, rRoute *routeBuilder, join *sqlparser.JoinTableExpr) (planBuilder, error) {
	if lRoute.Route.Keyspace.Name != rRoute.Route.Keyspace.Name {
		return makejoinBuilder(lRoute, rRoute, join)
	}
	if lRoute.Route.PlanID == SelectUnsharded {
		// Two Routes from the same unsharded keyspace can be merged.
		return mergeRoutes(lRoute, rRoute, join)
	}

	// TODO(sougou): Handle special case for SelectEqual

	// Both routeBuilder are sharded routes. Analyze join condition for merging.
	for _, filter := range splitAndExpression(nil, join.On) {
		if isSameRoute(lRoute, rRoute, filter) {
			return mergeRoutes(lRoute, rRoute, join)
		}
	}
	return makejoinBuilder(lRoute, rRoute, join)
}

// mergeRoutes makes a new routeBuilder by joining the left and right
// nodes of a join. The merged routeBuilder inherits the plan of the
// left Route. This function is called if two routes can be merged.
func mergeRoutes(lRoute, rRoute *routeBuilder, join *sqlparser.JoinTableExpr) (planBuilder, error) {
	lRoute.Select.From = sqlparser.TableExprs{join}
	if join.Join == sqlparser.LeftJoinStr {
		rRoute.Symtab().SetRHS()
	}
	err := lRoute.Symtab().Merge(rRoute.Symtab(), lRoute)
	if err != nil {
		return nil, err
	}
	for _, filter := range splitAndExpression(nil, join.On) {
		// If VTGate evolves, this section should be rewritten
		// to use processBoolExpr.
		_, err = findRoute(filter, lRoute)
		if err != nil {
			return nil, err
		}
		updateRoute(lRoute, filter)
	}
	return lRoute, nil
}

// isSameRoute returns true if the filter constraint causes the
// left and right routes to be part of the same route. For this
// to happen, the constraint has to be an equality like a.id = b.id,
// one should address a table from the left side, the other from the
// right, the referenced columns have to be the same Vindex, and the
// Vindex must be unique.
func isSameRoute(lRoute, rRoute *routeBuilder, filter sqlparser.BoolExpr) bool {
	comparison, ok := filter.(*sqlparser.ComparisonExpr)
	if !ok {
		return false
	}
	if comparison.Operator != sqlparser.EqualStr {
		return false
	}
	left := comparison.Left
	right := comparison.Right
	lVindex := lRoute.Symtab().Vindex(left, lRoute, false)
	if lVindex == nil {
		left, right = right, left
		lVindex = lRoute.Symtab().Vindex(left, lRoute, false)
	}
	if lVindex == nil || !IsUnique(lVindex) {
		return false
	}
	rVindex := rRoute.Symtab().Vindex(right, rRoute, false)
	if rVindex == nil {
		return false
	}
	if rVindex != lVindex {
		return false
	}
	return true
}
