// Copyright 2014, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/youtube/vitess/go/bson"
	"github.com/youtube/vitess/go/bytes2"
)

// GTID represents a Global Transaction ID, also known as Transaction Group ID.
// Each flavor of MySQL has its own format for the GTID. This interface is used
// along with various MysqlFlavor implementations to abstract the differences.
//
// Types that implement GTID should use a non-pointer receiver. This ensures
// that comparing GTID interface values with == has the expected semantics.
type GTID interface {
	// String returns the canonical form of the GTID as expected by a particular
	// flavor of MySQL.
	String() string

	// Flavor returns the key under which the corresponding GTID parser function
	// is registered in the GTIDParsers map.
	Flavor() string

	// TryCompare tries to compare two GTIDs. Some flavors of GTID can always be
	// compared (e.g. Google MySQL group_id). Others can only be compared if they
	// came from the same master (e.g. MariaDB, MySQL 5.6).
	//
	// If the comparison is possible, a.TryCompare(b) will return an int that is:
	//    < 0  if a < b  (a came before b)
	//   == 0  if a == b
	//    > 0  if a > b  (a came after b)
	//
	// If the comparison is not possible, a non-nil error will be returned.
	TryCompare(GTID) (int, error)
}

// gtidParsers maps flavor names to parser functions.
var gtidParsers = make(map[string]func(string) (GTID, error))

// ParseGTID calls the GTID parser for the specified flavor.
func ParseGTID(flavor, value string) (GTID, error) {
	parser := gtidParsers[flavor]
	if parser == nil {
		return nil, fmt.Errorf("ParseGTID: unknown flavor '%v'", flavor)
	}
	return parser(value)
}

// MustParseGTID calls ParseGTID and panics on error.
func MustParseGTID(flavor, value string) GTID {
	gtid, err := ParseGTID(flavor, value)
	if err != nil {
		panic(err)
	}
	return gtid
}

// EncodeGTID returns a string that contains both the flavor and value of the
// GTID, so that the correct parser can be selected when that string is passed
// to DecodeGTID.
func EncodeGTID(gtid GTID) string {
	if gtid == nil {
		return ""
	}

	return fmt.Sprintf("%s/%s", gtid.Flavor(), gtid.String())
}

// DecodeGTID converts a string in the format returned by EncodeGTID back into
// a GTID interface value with the correct underlying flavor.
func DecodeGTID(s string) (GTID, error) {
	if s == "" {
		return nil, nil
	}

	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		// There is no flavor. Try looking for a default parser.
		parts = []string{"", s}
	}
	return ParseGTID(parts[0], parts[1])
}

// GTIDField is a concrete struct that contains a GTID interface value. This can
// be used as a field inside marshalable structs, which cannot contain interface
// values because there would be no way to know which concrete type to
// instantiate upon unmarshaling.
type GTIDField struct {
	GTID
}

// String returns a string representation of the underlying GTID.
func (gf GTIDField) String() string {
	if gf.GTID == nil {
		return "<nil>" // in the style of Sprintf("%v", nil)
	}
	return gf.GTID.String()
}

// MarshalBson bson-encodes GTIDField.
func (gf GTIDField) MarshalBson(buf *bytes2.ChunkedWriter, key string) {
	bson.EncodeOptionalPrefix(buf, bson.Object, key)

	lenWriter := bson.NewLenWriter(buf)

	if gf.GTID != nil {
		// The name of the bson field is the MySQL flavor.
		bson.EncodeString(buf, gf.GTID.Flavor(), gf.GTID.String())
	}

	lenWriter.Close()
}

// UnmarshalBson bson-decodes into GTIDField.
func (gf *GTIDField) UnmarshalBson(buf *bytes.Buffer, kind byte) {
	switch kind {
	case bson.EOO, bson.Object:
		// valid
	case bson.Null:
		return
	default:
		panic(bson.NewBsonError("unexpected kind %v for GTIDField", kind))
	}
	bson.Next(buf, 4)

	// We expect exactly zero or one fields in this bson object.
	kind = bson.NextByte(buf)
	if kind == bson.EOO {
		// The GTID was nil, nothing to do.
		return
	}

	// The field name is the MySQL flavor.
	flavor := bson.ReadCString(buf)
	value := bson.DecodeString(buf, kind)

	// Check for and consume the end byte.
	if kind = bson.NextByte(buf); kind != bson.EOO {
		panic(bson.NewBsonError("too many fields for GTIDField"))
	}

	// Parse the value.
	gtid, err := ParseGTID(flavor, value)
	if err != nil {
		panic(bson.NewBsonError("invalid value %v for GTIDField: %v", value, err))
	}
	gf.GTID = gtid
}

// MarshalJSON implements encoding/json.Marshaler.
func (gf GTIDField) MarshalJSON() ([]byte, error) {
	return json.Marshal(EncodeGTID(gf.GTID))
}

// UnmarshalJSON implements encoding/json.Unmarshaler.
func (gf *GTIDField) UnmarshalJSON(buf []byte) error {
	var s string
	err := json.Unmarshal(buf, &s)
	if err != nil {
		return err
	}

	gf.GTID, err = DecodeGTID(s)
	if err != nil {
		return err
	}
	return nil
}
