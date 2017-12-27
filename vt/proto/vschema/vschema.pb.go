// Code generated by protoc-gen-go. DO NOT EDIT.
// source: vschema.proto

/*
Package vschema is a generated protocol buffer package.

It is generated from these files:
	vschema.proto

It has these top-level messages:
	Keyspace
	Vindex
	Table
	ColumnVindex
	AutoIncrement
	Column
	SrvVSchema
*/
package vschema

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import query "github.com/youtube/vitess/go/vt/proto/query"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Keyspace is the vschema for a keyspace.
type Keyspace struct {
	// If sharded is false, vindexes and tables are ignored.
	Sharded  bool               `protobuf:"varint,1,opt,name=sharded" json:"sharded,omitempty"`
	Vindexes map[string]*Vindex `protobuf:"bytes,2,rep,name=vindexes" json:"vindexes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Tables   map[string]*Table  `protobuf:"bytes,3,rep,name=tables" json:"tables,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *Keyspace) Reset()                    { *m = Keyspace{} }
func (m *Keyspace) String() string            { return proto.CompactTextString(m) }
func (*Keyspace) ProtoMessage()               {}
func (*Keyspace) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Keyspace) GetSharded() bool {
	if m != nil {
		return m.Sharded
	}
	return false
}

func (m *Keyspace) GetVindexes() map[string]*Vindex {
	if m != nil {
		return m.Vindexes
	}
	return nil
}

func (m *Keyspace) GetTables() map[string]*Table {
	if m != nil {
		return m.Tables
	}
	return nil
}

// Vindex is the vindex info for a Keyspace.
type Vindex struct {
	// The type must match one of the predefined
	// (or plugged in) vindex names.
	Type string `protobuf:"bytes,1,opt,name=type" json:"type,omitempty"`
	// params is a map of attribute value pairs
	// that must be defined as required by the
	// vindex constructors. The values can only
	// be strings.
	Params map[string]string `protobuf:"bytes,2,rep,name=params" json:"params,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// A lookup vindex can have an owner table defined.
	// If so, rows in the lookup table are created or
	// deleted in sync with corresponding rows in the
	// owner table.
	Owner string `protobuf:"bytes,3,opt,name=owner" json:"owner,omitempty"`
}

func (m *Vindex) Reset()                    { *m = Vindex{} }
func (m *Vindex) String() string            { return proto.CompactTextString(m) }
func (*Vindex) ProtoMessage()               {}
func (*Vindex) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *Vindex) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *Vindex) GetParams() map[string]string {
	if m != nil {
		return m.Params
	}
	return nil
}

func (m *Vindex) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

// Table is the table info for a Keyspace.
type Table struct {
	// If the table is a sequence, type must be
	// "sequence". Otherwise, it should be empty.
	Type string `protobuf:"bytes,1,opt,name=type" json:"type,omitempty"`
	// column_vindexes associates columns to vindexes.
	ColumnVindexes []*ColumnVindex `protobuf:"bytes,2,rep,name=column_vindexes,json=columnVindexes" json:"column_vindexes,omitempty"`
	// auto_increment is specified if a column needs
	// to be associated with a sequence.
	AutoIncrement *AutoIncrement `protobuf:"bytes,3,opt,name=auto_increment,json=autoIncrement" json:"auto_increment,omitempty"`
	// columns lists the columns for the table.
	Columns []*Column `protobuf:"bytes,4,rep,name=columns" json:"columns,omitempty"`
}

func (m *Table) Reset()                    { *m = Table{} }
func (m *Table) String() string            { return proto.CompactTextString(m) }
func (*Table) ProtoMessage()               {}
func (*Table) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *Table) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *Table) GetColumnVindexes() []*ColumnVindex {
	if m != nil {
		return m.ColumnVindexes
	}
	return nil
}

func (m *Table) GetAutoIncrement() *AutoIncrement {
	if m != nil {
		return m.AutoIncrement
	}
	return nil
}

func (m *Table) GetColumns() []*Column {
	if m != nil {
		return m.Columns
	}
	return nil
}

// ColumnVindex is used to associate a column to a vindex.
type ColumnVindex struct {
	Column string `protobuf:"bytes,1,opt,name=column" json:"column,omitempty"`
	// The name must match a vindex defined in Keyspace.
	Name string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
}

func (m *ColumnVindex) Reset()                    { *m = ColumnVindex{} }
func (m *ColumnVindex) String() string            { return proto.CompactTextString(m) }
func (*ColumnVindex) ProtoMessage()               {}
func (*ColumnVindex) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *ColumnVindex) GetColumn() string {
	if m != nil {
		return m.Column
	}
	return ""
}

func (m *ColumnVindex) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

// Autoincrement is used to designate a column as auto-inc.
type AutoIncrement struct {
	Column string `protobuf:"bytes,1,opt,name=column" json:"column,omitempty"`
	// The sequence must match a table of type SEQUENCE.
	Sequence string `protobuf:"bytes,2,opt,name=sequence" json:"sequence,omitempty"`
}

func (m *AutoIncrement) Reset()                    { *m = AutoIncrement{} }
func (m *AutoIncrement) String() string            { return proto.CompactTextString(m) }
func (*AutoIncrement) ProtoMessage()               {}
func (*AutoIncrement) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *AutoIncrement) GetColumn() string {
	if m != nil {
		return m.Column
	}
	return ""
}

func (m *AutoIncrement) GetSequence() string {
	if m != nil {
		return m.Sequence
	}
	return ""
}

// Column describes a column.
type Column struct {
	Name string     `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Type query.Type `protobuf:"varint,2,opt,name=type,enum=query.Type" json:"type,omitempty"`
}

func (m *Column) Reset()                    { *m = Column{} }
func (m *Column) String() string            { return proto.CompactTextString(m) }
func (*Column) ProtoMessage()               {}
func (*Column) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *Column) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Column) GetType() query.Type {
	if m != nil {
		return m.Type
	}
	return query.Type_NULL_TYPE
}

// SrvVSchema is the roll-up of all the Keyspace schema for a cell.
type SrvVSchema struct {
	// keyspaces is a map of keyspace name -> Keyspace object.
	Keyspaces map[string]*Keyspace `protobuf:"bytes,1,rep,name=keyspaces" json:"keyspaces,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *SrvVSchema) Reset()                    { *m = SrvVSchema{} }
func (m *SrvVSchema) String() string            { return proto.CompactTextString(m) }
func (*SrvVSchema) ProtoMessage()               {}
func (*SrvVSchema) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *SrvVSchema) GetKeyspaces() map[string]*Keyspace {
	if m != nil {
		return m.Keyspaces
	}
	return nil
}

func init() {
	proto.RegisterType((*Keyspace)(nil), "vschema.Keyspace")
	proto.RegisterType((*Vindex)(nil), "vschema.Vindex")
	proto.RegisterType((*Table)(nil), "vschema.Table")
	proto.RegisterType((*ColumnVindex)(nil), "vschema.ColumnVindex")
	proto.RegisterType((*AutoIncrement)(nil), "vschema.AutoIncrement")
	proto.RegisterType((*Column)(nil), "vschema.Column")
	proto.RegisterType((*SrvVSchema)(nil), "vschema.SrvVSchema")
}

func init() { proto.RegisterFile("vschema.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 484 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x54, 0xdb, 0x6e, 0xd3, 0x40,
	0x10, 0xd5, 0x26, 0x8d, 0x93, 0x8c, 0x89, 0x03, 0xab, 0x52, 0x59, 0x46, 0xa8, 0x91, 0x05, 0x22,
	0xbc, 0xf8, 0x21, 0x15, 0x12, 0x14, 0x15, 0x81, 0x22, 0x1e, 0x2a, 0x90, 0x40, 0x6e, 0xd5, 0xd7,
	0x6a, 0xeb, 0x8c, 0xd4, 0xaa, 0xf1, 0xa5, 0x5e, 0x3b, 0xe0, 0xaf, 0x41, 0xe2, 0x0f, 0xf8, 0x08,
	0xfe, 0x0b, 0x65, 0x6f, 0x59, 0x27, 0xe1, 0x6d, 0x27, 0x67, 0xce, 0x99, 0x33, 0x93, 0x19, 0xc3,
	0x68, 0xc5, 0x93, 0x5b, 0x4c, 0x59, 0x54, 0x94, 0x79, 0x95, 0xd3, 0xbe, 0x0a, 0x03, 0xf7, 0xa1,
	0xc6, 0xb2, 0x91, 0xbf, 0x86, 0x7f, 0x3a, 0x30, 0xf8, 0x82, 0x0d, 0x2f, 0x58, 0x82, 0xd4, 0x87,
	0x3e, 0xbf, 0x65, 0xe5, 0x02, 0x17, 0x3e, 0x99, 0x90, 0xe9, 0x20, 0xd6, 0x21, 0x7d, 0x0f, 0x83,
	0xd5, 0x5d, 0xb6, 0xc0, 0x9f, 0xc8, 0xfd, 0xce, 0xa4, 0x3b, 0x75, 0x67, 0xc7, 0x91, 0x96, 0xd7,
	0xf4, 0xe8, 0x4a, 0x65, 0x7c, 0xce, 0xaa, 0xb2, 0x89, 0x0d, 0x81, 0xbe, 0x01, 0xa7, 0x62, 0x37,
	0x4b, 0xe4, 0x7e, 0x57, 0x50, 0x9f, 0xef, 0x52, 0x2f, 0x05, 0x2e, 0x89, 0x2a, 0x39, 0xf8, 0x0a,
	0xa3, 0x96, 0x22, 0x7d, 0x0c, 0xdd, 0x7b, 0x6c, 0x84, 0xb5, 0x61, 0xbc, 0x7e, 0xd2, 0x97, 0xd0,
	0x5b, 0xb1, 0x65, 0x8d, 0x7e, 0x67, 0x42, 0xa6, 0xee, 0x6c, 0x6c, 0x84, 0x25, 0x31, 0x96, 0xe8,
	0x69, 0xe7, 0x2d, 0x09, 0xce, 0xc1, 0xb5, 0x8a, 0xec, 0xd1, 0x7a, 0xd1, 0xd6, 0xf2, 0x8c, 0x96,
	0xa0, 0x59, 0x52, 0xe1, 0x6f, 0x02, 0x8e, 0x2c, 0x40, 0x29, 0x1c, 0x54, 0x4d, 0x81, 0x4a, 0x47,
	0xbc, 0xe9, 0x09, 0x38, 0x05, 0x2b, 0x59, 0xaa, 0x27, 0xf5, 0x6c, 0xcb, 0x55, 0xf4, 0x5d, 0xa0,
	0xaa, 0x59, 0x99, 0x4a, 0x0f, 0xa1, 0x97, 0xff, 0xc8, 0xb0, 0xf4, 0xbb, 0x42, 0x49, 0x06, 0xc1,
	0x3b, 0x70, 0xad, 0xe4, 0x3d, 0xa6, 0x0f, 0x6d, 0xd3, 0x43, 0xdb, 0xe4, 0x5f, 0x02, 0x3d, 0xe1,
	0x7c, 0xaf, 0xc7, 0x0f, 0x30, 0x4e, 0xf2, 0x65, 0x9d, 0x66, 0xd7, 0x5b, 0x7f, 0xeb, 0x53, 0x63,
	0x76, 0x2e, 0x70, 0x35, 0x48, 0x2f, 0xb1, 0x22, 0xe4, 0xf4, 0x0c, 0x3c, 0x56, 0x57, 0xf9, 0xf5,
	0x5d, 0x96, 0x94, 0x98, 0x62, 0x56, 0x09, 0xdf, 0xee, 0xec, 0xc8, 0xd0, 0x3f, 0xd5, 0x55, 0x7e,
	0xae, 0xd1, 0x78, 0xc4, 0xec, 0x90, 0xbe, 0x86, 0xbe, 0x14, 0xe4, 0xfe, 0x81, 0x28, 0x3b, 0xde,
	0x2a, 0x1b, 0x6b, 0x3c, 0x3c, 0x85, 0x47, 0xb6, 0x13, 0x7a, 0x04, 0x8e, 0x84, 0x54, 0x3f, 0x2a,
	0x5a, 0x77, 0x99, 0xb1, 0x54, 0x0f, 0x42, 0xbc, 0xc3, 0x39, 0x8c, 0x5a, 0x36, 0xfe, 0x4b, 0x0e,
	0x60, 0xc0, 0xf1, 0xa1, 0xc6, 0x2c, 0xd1, 0x02, 0x26, 0x0e, 0xcf, 0xc0, 0x99, 0xb7, 0x4b, 0x90,
	0x4d, 0x09, 0x7a, 0xac, 0x86, 0xbb, 0x66, 0x79, 0x33, 0x37, 0x92, 0xb7, 0x75, 0xd9, 0x14, 0x28,
	0x27, 0x1d, 0xfe, 0x22, 0x00, 0x17, 0xe5, 0xea, 0xea, 0x42, 0xb4, 0x47, 0x3f, 0xc2, 0xf0, 0x5e,
	0x2d, 0x3d, 0xf7, 0x89, 0xe8, 0x3d, 0x34, 0xbd, 0x6f, 0xf2, 0xcc, 0x65, 0xa8, 0x35, 0xd9, 0x90,
	0x82, 0x6f, 0xe0, 0xb5, 0xc1, 0x3d, 0x6b, 0xf1, 0xaa, 0xbd, 0xcb, 0x4f, 0x76, 0x0e, 0xce, 0xda,
	0x94, 0x1b, 0x47, 0x7c, 0x09, 0x4e, 0xfe, 0x05, 0x00, 0x00, 0xff, 0xff, 0xd5, 0x7c, 0x38, 0x52,
	0x30, 0x04, 0x00, 0x00,
}
