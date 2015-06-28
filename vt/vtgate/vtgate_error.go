// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vtgate

import (
	mproto "github.com/youtube/vitess/go/mysql/proto"
	"github.com/youtube/vitess/go/vt/vterrors"
	"github.com/youtube/vitess/go/vt/vtgate/proto"
)

// rpcErrFromTabletError translate an error from VTGate to an *mproto.RPCError
func rpcErrFromVtGateError(err error) *mproto.RPCError {
	if err == nil {
		return nil
	}
	// TODO(aaijazi): for now, we don't have any differentiation of VtGate errors.
	// However, we should have them soon, so that clients don't have to parse the
	// returned error string.
	return &mproto.RPCError{
		Code:    vterrors.UnknownVtgateError,
		Message: err.Error(),
	}
}

// AddVtGateErrorToQueryResult will mutate a QueryResult struct to fill in the Err
// field with details from the VTGate error.
func AddVtGateErrorToQueryResult(err error, reply *proto.QueryResult) {
	if err == nil {
		return
	}
	reply.Err = rpcErrFromVtGateError(err)
}
