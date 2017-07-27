/*
Copyright 2017 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sqltypes

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	querypb "github.com/youtube/vitess/go/vt/proto/query"
)

// numeric represents a numeric value extracted from
// a Value, used for arithmetic operations.
type numeric struct {
	typ  querypb.Type
	ival int64
	uval uint64
	fval float64
}

// NullsafeAdd adds two Values in a null-safe manner. A null value
// is treated as 0. If both values are null, then a null is returned.
// If both values are not null, a numeric value is built
// from each input: Signed->int64, Unsigned->uint64, Float->float64.
// Otherwise the 'best type fit' is chosen for the number: int64 or float64.
// Addition is performed by upgrading types as needed, or in case
// of overflow: int64->uint64, int64->float64, uint64->float64.
// Unsigned ints can only be added to positive ints. After the
// addition, if one of the input types was Decimal, then
// a Decimal is built. Otherwise, the final type of the
// result is preserved.
func NullsafeAdd(v1, v2 Value, resultType querypb.Type) (Value, error) {
	if v1.IsNull() {
		return v2, nil
	}
	if v2.IsNull() {
		return v1, nil
	}

	lv1, err := newNumeric(v1)
	if err != nil {
		return NULL, err
	}
	lv2, err := newNumeric(v2)
	if err != nil {
		return NULL, err
	}
	lresult, err := addNumeric(lv1, lv2)
	if err != nil {
		return NULL, err
	}
	return castFromNumeric(lresult, resultType)
}

// NullsafeCompare returns 0 if v1==v2, -1 if v1<v2, and 1 if v1>v2.
// NULL is the lowest value. If any value is
// numeric, then a numeric comparison is performed after
// necessary conversions. If none are numeric, then it's
// a simple binary comparison. Text values return an error.
func NullsafeCompare(v1, v2 Value) (int, error) {
	if v1.IsNull() {
		if v2.IsNull() {
			return 0, nil
		}
		return -1, nil
	}
	if v2.IsNull() {
		return 1, nil
	}
	if v1.IsText() || v2.IsText() {
		return 0, errors.New("text fields cannot be compared")
	}
	if isNumber(v1.Type()) || isNumber(v2.Type()) {
		lv1, err := newNumeric(v1)
		if err != nil {
			return 0, err
		}
		lv2, err := newNumeric(v2)
		if err != nil {
			return 0, err
		}
		return compareNumeric(lv1, lv2), nil
	}
	// TODO(sougou): perform a more type-aware comparison instead.
	return bytes.Compare(v1.Raw(), v2.Raw()), nil
}

// Min returns the minimum of v1 and v2. If one of the
// values is NULL, it returns the other value. If both
// are NULL, it returns NULL.
func Min(v1, v2 Value) (Value, error) {
	return minmax(v1, v2, true)
}

// Max returns the maximum of v1 and v2. If one of the
// values is NULL, it returns the other value. If both
// are NULL, it returns NULL.
func Max(v1, v2 Value) (Value, error) {
	return minmax(v1, v2, false)
}

func minmax(v1, v2 Value, min bool) (Value, error) {
	if v1.IsNull() {
		return v2, nil
	}
	if v2.IsNull() {
		return v1, nil
	}

	n, err := NullsafeCompare(v1, v2)
	if err != nil {
		return NULL, err
	}

	// XNOR construct. See tests.
	v1isSmaller := n < 0
	if min == v1isSmaller {
		return v1, nil
	}
	return v2, nil
}

// Cast converts a Value to the target type.
func Cast(v Value, typ querypb.Type) (Value, error) {
	if v.Type() == typ || v.IsNull() {
		return v, nil
	}
	if IsSigned(typ) && v.IsSigned() {
		return MakeTrusted(typ, v.Raw()), nil
	}
	if IsUnsigned(typ) && v.IsUnsigned() {
		return MakeTrusted(typ, v.Raw()), nil
	}
	if (IsFloat(typ) || typ == Decimal) && (v.IsIntegral() || v.IsFloat() || v.Type() == Decimal) {
		return MakeTrusted(typ, v.Raw()), nil
	}
	if IsQuoted(typ) && (v.IsIntegral() || v.IsFloat() || v.Type() == Decimal || v.IsQuoted()) {
		return MakeTrusted(typ, v.Raw()), nil
	}

	// Explicitly disallow Expression.
	if v.Type() == Expression {
		return NULL, fmt.Errorf("%v value cannot be cast to %v", v.Type(), typ)
	}

	// If the above fast-paths were not possible,
	// go through full validation.
	return NewValue(typ, v.Raw())
}

// ToUint64 converts Value to uint64.
func ToUint64(v Value) (uint64, error) {
	num, err := newIntegralNumeric(v)
	if err != nil {
		return 0, err
	}
	switch num.typ {
	case Int64:
		if num.ival < 0 {
			return 0, fmt.Errorf("negative number cannot be converted to unsigned: %d", num.ival)
		}
		return uint64(num.ival), nil
	case Uint64:
		return num.uval, nil
	}
	panic("unreachable")
}

// ToInt64 converts Value to uint64.
func ToInt64(v Value) (int64, error) {
	num, err := newIntegralNumeric(v)
	if err != nil {
		return 0, err
	}
	switch num.typ {
	case Int64:
		return num.ival, nil
	case Uint64:
		ival := int64(num.uval)
		if ival < 0 {
			return 0, fmt.Errorf("unsigned number overflows int64 value: %d", num.uval)
		}
		return ival, nil
	}
	panic("unreachable")
}

// ToFloat64 converts Value to float64.
func ToFloat64(v Value) (float64, error) {
	num, err := newNumeric(v)
	if err != nil {
		return 0, err
	}
	switch num.typ {
	case Int64:
		return float64(num.ival), nil
	case Uint64:
		return float64(num.uval), nil
	case Float64:
		return num.fval, nil
	}
	panic("unreachable")
}

// ToNative converts Value to a native go type.
// Decimal is returned as []byte.
func ToNative(v Value) (interface{}, error) {
	var out interface{}
	var err error
	switch {
	case v.Type() == Null:
		// no-op
	case IsSigned(v.Type()):
		return ToInt64(v)
	case IsUnsigned(v.Type()):
		return ToUint64(v)
	case IsFloat(v.Type()):
		return ToFloat64(v)
	case IsQuoted(v.Type()) || v.Type() == Decimal:
		out = v.val
	case v.Type() == Expression:
		err = errors.New("EXPRESSION cannot be converted to a go type")
	}
	return out, err
}

// newNumeric parses a value and produces an Int64, Uint64 or Float64.
func newNumeric(v Value) (result numeric, err error) {
	str := v.String()
	switch {
	case v.IsSigned():
		result.ival, err = strconv.ParseInt(str, 10, 64)
		result.typ = Int64
		return
	case v.IsUnsigned():
		result.uval, err = strconv.ParseUint(str, 10, 64)
		result.typ = Uint64
		return
	case v.IsFloat():
		result.fval, err = strconv.ParseFloat(str, 64)
		result.typ = Float64
		return
	}

	// For other types, do best effort.
	result.ival, err = strconv.ParseInt(str, 10, 64)
	if err == nil {
		result.typ = Int64
		return
	}
	result.fval, err = strconv.ParseFloat(str, 64)
	if err == nil {
		result.typ = Float64
		return
	}
	err = fmt.Errorf("could not parse value: %s", str)
	return
}

// newIntegralNumeric parses a value and produces an Int64 or Uint64.
func newIntegralNumeric(v Value) (result numeric, err error) {
	str := v.String()
	switch {
	case v.IsSigned():
		result.ival, err = strconv.ParseInt(str, 10, 64)
		if err != nil {
			return
		}
		result.typ = Int64
		return
	case v.IsUnsigned():
		result.uval, err = strconv.ParseUint(str, 10, 64)
		if err != nil {
			return
		}
		result.typ = Uint64
		return
	}

	// For other types, do best effort.
	result.ival, err = strconv.ParseInt(str, 10, 64)
	if err == nil {
		result.typ = Int64
		return
	}
	// ParseInt can return a non-zero value on failure.
	result.ival = 0
	result.uval, err = strconv.ParseUint(str, 10, 64)
	if err == nil {
		result.typ = Uint64
		return
	}
	err = fmt.Errorf("could not parse value: %s", str)
	return
}

func addNumeric(v1, v2 numeric) (numeric, error) {
	v1, v2 = prioritize(v1, v2)
	switch v1.typ {
	case Int64:
		return intPlusInt(v1.ival, v2.ival), nil
	case Uint64:
		switch v2.typ {
		case Int64:
			return uintPlusInt(v1.uval, v2.ival)
		case Uint64:
			return uintPlusUint(v1.uval, v2.uval), nil
		}
	case Float64:
		return floatPlusAny(v1.fval, v2), nil
	}
	panic("unreachable")
}

// prioritize reorders the input parameters
// to be Float64, Uint64, Int64.
func prioritize(v1, v2 numeric) (altv1, altv2 numeric) {
	switch v1.typ {
	case Int64:
		if v2.typ == Uint64 || v2.typ == Float64 {
			return v2, v1
		}
	case Uint64:
		if v2.typ == Float64 {
			return v2, v1
		}
	}
	return v1, v2
}

func intPlusInt(v1, v2 int64) numeric {
	result := v1 + v2
	if v1 > 0 && v2 > 0 && result < 0 {
		goto overflow
	}
	if v1 < 0 && v2 < 0 && result > 0 {
		goto overflow
	}
	return numeric{typ: Int64, ival: result}

overflow:
	return numeric{typ: Float64, fval: float64(v1) + float64(v2)}
}

func uintPlusInt(v1 uint64, v2 int64) (numeric, error) {
	if v2 < 0 {
		return numeric{}, fmt.Errorf("cannot add a negative number to an unsigned integer: %d, %d", v1, v2)
	}
	return uintPlusUint(v1, uint64(v2)), nil
}

func uintPlusUint(v1, v2 uint64) numeric {
	result := v1 + v2
	if result < v2 {
		return numeric{typ: Float64, fval: float64(v1) + float64(v2)}
	}
	return numeric{typ: Uint64, uval: result}
}

func floatPlusAny(v1 float64, v2 numeric) numeric {
	switch v2.typ {
	case Int64:
		v2.fval = float64(v2.ival)
	case Uint64:
		v2.fval = float64(v2.uval)
	}
	return numeric{typ: Float64, fval: v1 + v2.fval}
}

func castFromNumeric(v numeric, resultType querypb.Type) (Value, error) {
	switch {
	case IsSigned(resultType):
		switch v.typ {
		case Int64:
			return MakeTrusted(resultType, strconv.AppendInt(nil, v.ival, 10)), nil
		case Uint64, Float64:
			return NULL, fmt.Errorf("unexpected type conversion: %v to %v", v.typ, resultType)
		}
	case IsUnsigned(resultType):
		switch v.typ {
		case Uint64:
			return MakeTrusted(resultType, strconv.AppendUint(nil, v.uval, 10)), nil
		case Int64, Float64:
			return NULL, fmt.Errorf("unexpected type conversion: %v to %v", v.typ, resultType)
		}
	case IsFloat(resultType) || resultType == Decimal:
		switch v.typ {
		case Int64:
			return MakeTrusted(resultType, strconv.AppendInt(nil, v.ival, 10)), nil
		case Uint64:
			return MakeTrusted(resultType, strconv.AppendUint(nil, v.uval, 10)), nil
		case Float64:
			format := byte('g')
			if resultType == Decimal {
				format = 'f'
			}
			return MakeTrusted(resultType, strconv.AppendFloat(nil, v.fval, format, -1, 64)), nil
		}
	}
	return NULL, fmt.Errorf("unexpected type conversion to non-numeric: %v", resultType)
}

func compareNumeric(v1, v2 numeric) int {
	// Equalize the types.
	switch v1.typ {
	case Int64:
		switch v2.typ {
		case Uint64:
			if v1.ival < 0 {
				return -1
			}
			v1 = numeric{typ: Uint64, uval: uint64(v1.ival)}
		case Float64:
			v1 = numeric{typ: Float64, fval: float64(v1.ival)}
		}
	case Uint64:
		switch v2.typ {
		case Int64:
			if v2.ival < 0 {
				return 1
			}
			v2 = numeric{typ: Uint64, uval: uint64(v2.ival)}
		case Float64:
			v1 = numeric{typ: Float64, fval: float64(v1.uval)}
		}
	case Float64:
		switch v2.typ {
		case Int64:
			v2 = numeric{typ: Float64, fval: float64(v2.ival)}
		case Uint64:
			v2 = numeric{typ: Float64, fval: float64(v2.uval)}
		}
	}

	// Both values are of the same type.
	switch v1.typ {
	case Int64:
		switch {
		case v1.ival == v2.ival:
			return 0
		case v1.ival < v2.ival:
			return -1
		}
	case Uint64:
		switch {
		case v1.uval == v2.uval:
			return 0
		case v1.uval < v2.uval:
			return -1
		}
	case Float64:
		switch {
		case v1.fval == v2.fval:
			return 0
		case v1.fval < v2.fval:
			return -1
		}
	}

	// v1>v2
	return 1
}
