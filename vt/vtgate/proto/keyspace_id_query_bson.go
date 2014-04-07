// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

// DO NOT EDIT.
// FILE GENERATED BY BSONGEN.

import (
	"bytes"

	"github.com/youtube/vitess/go/bson"
	"github.com/youtube/vitess/go/bytes2"
	kproto "github.com/youtube/vitess/go/vt/key"
)

// MarshalBson bson-encodes KeyspaceIdQuery.
func (keyspaceIdQuery *KeyspaceIdQuery) MarshalBson(buf *bytes2.ChunkedWriter, key string) {
	bson.EncodeOptionalPrefix(buf, bson.Object, key)
	lenWriter := bson.NewLenWriter(buf)

	bson.EncodeString(buf, "Sql", keyspaceIdQuery.Sql)
	// map[string]interface{}
	{
		bson.EncodePrefix(buf, bson.Object, "BindVariables")
		lenWriter := bson.NewLenWriter(buf)
		for _k, _v1 := range keyspaceIdQuery.BindVariables {
			bson.EncodeInterface(buf, _k, _v1)
		}
		lenWriter.Close()
	}
	bson.EncodeString(buf, "Keyspace", keyspaceIdQuery.Keyspace)
	// []kproto.KeyspaceId
	{
		bson.EncodePrefix(buf, bson.Array, "KeyspaceIds")
		lenWriter := bson.NewLenWriter(buf)
		for _i, _v2 := range keyspaceIdQuery.KeyspaceIds {
			_v2.MarshalBson(buf, bson.Itoa(_i))
		}
		lenWriter.Close()
	}
	keyspaceIdQuery.TabletType.MarshalBson(buf, "TabletType")
	// *Session
	if keyspaceIdQuery.Session == nil {
		bson.EncodePrefix(buf, bson.Null, "Session")
	} else {
		(*keyspaceIdQuery.Session).MarshalBson(buf, "Session")
	}

	lenWriter.Close()
}

// UnmarshalBson bson-decodes into KeyspaceIdQuery.
func (keyspaceIdQuery *KeyspaceIdQuery) UnmarshalBson(buf *bytes.Buffer, kind byte) {
	switch kind {
	case bson.EOO, bson.Object:
		// valid
	case bson.Null:
		return
	default:
		panic(bson.NewBsonError("unexpected kind %v for KeyspaceIdQuery", kind))
	}
	bson.Next(buf, 4)

	for kind := bson.NextByte(buf); kind != bson.EOO; kind = bson.NextByte(buf) {
		switch bson.ReadCString(buf) {
		case "Sql":
			keyspaceIdQuery.Sql = bson.DecodeString(buf, kind)
		case "BindVariables":
			// map[string]interface{}
			if kind != bson.Null {
				if kind != bson.Object {
					panic(bson.NewBsonError("unexpected kind %v for keyspaceIdQuery.BindVariables", kind))
				}
				bson.Next(buf, 4)
				keyspaceIdQuery.BindVariables = make(map[string]interface{})
				for kind := bson.NextByte(buf); kind != bson.EOO; kind = bson.NextByte(buf) {
					_k := bson.ReadCString(buf)
					var _v1 interface{}
					_v1 = bson.DecodeInterface(buf, kind)
					keyspaceIdQuery.BindVariables[_k] = _v1
				}
			}
		case "Keyspace":
			keyspaceIdQuery.Keyspace = bson.DecodeString(buf, kind)
		case "KeyspaceIds":
			// []kproto.KeyspaceId
			if kind != bson.Null {
				if kind != bson.Array {
					panic(bson.NewBsonError("unexpected kind %v for keyspaceIdQuery.KeyspaceIds", kind))
				}
				bson.Next(buf, 4)
				keyspaceIdQuery.KeyspaceIds = make([]kproto.KeyspaceId, 0, 8)
				for kind := bson.NextByte(buf); kind != bson.EOO; kind = bson.NextByte(buf) {
					bson.SkipIndex(buf)
					var _v2 kproto.KeyspaceId
					_v2.UnmarshalBson(buf, kind)
					keyspaceIdQuery.KeyspaceIds = append(keyspaceIdQuery.KeyspaceIds, _v2)
				}
			}
		case "TabletType":
			keyspaceIdQuery.TabletType.UnmarshalBson(buf, kind)
		case "Session":
			// *Session
			if kind != bson.Null {
				keyspaceIdQuery.Session = new(Session)
				(*keyspaceIdQuery.Session).UnmarshalBson(buf, kind)
			}
		default:
			bson.Skip(buf, kind)
		}
	}
}
