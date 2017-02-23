// Code generated by protoc-gen-go.
// source: binlogdata.proto
// DO NOT EDIT!

/*
Package binlogdata is a generated protocol buffer package.

It is generated from these files:
	binlogdata.proto

It has these top-level messages:
	Charset
	BinlogTransaction
	StreamKeyRangeRequest
	StreamKeyRangeResponse
	StreamTablesRequest
	StreamTablesResponse
*/
package binlogdata

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import query "github.com/youtube/vitess/go/vt/proto/query"
import topodata "github.com/youtube/vitess/go/vt/proto/topodata"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type BinlogTransaction_Statement_Category int32

const (
	BinlogTransaction_Statement_BL_UNRECOGNIZED BinlogTransaction_Statement_Category = 0
	BinlogTransaction_Statement_BL_BEGIN        BinlogTransaction_Statement_Category = 1
	BinlogTransaction_Statement_BL_COMMIT       BinlogTransaction_Statement_Category = 2
	BinlogTransaction_Statement_BL_ROLLBACK     BinlogTransaction_Statement_Category = 3
	// BL_DML is deprecated.
	BinlogTransaction_Statement_BL_DML_DEPRECATED BinlogTransaction_Statement_Category = 4
	BinlogTransaction_Statement_BL_DDL            BinlogTransaction_Statement_Category = 5
	BinlogTransaction_Statement_BL_SET            BinlogTransaction_Statement_Category = 6
	BinlogTransaction_Statement_BL_INSERT         BinlogTransaction_Statement_Category = 7
	BinlogTransaction_Statement_BL_UPDATE         BinlogTransaction_Statement_Category = 8
	BinlogTransaction_Statement_BL_DELETE         BinlogTransaction_Statement_Category = 9
)

var BinlogTransaction_Statement_Category_name = map[int32]string{
	0: "BL_UNRECOGNIZED",
	1: "BL_BEGIN",
	2: "BL_COMMIT",
	3: "BL_ROLLBACK",
	4: "BL_DML_DEPRECATED",
	5: "BL_DDL",
	6: "BL_SET",
	7: "BL_INSERT",
	8: "BL_UPDATE",
	9: "BL_DELETE",
}
var BinlogTransaction_Statement_Category_value = map[string]int32{
	"BL_UNRECOGNIZED":   0,
	"BL_BEGIN":          1,
	"BL_COMMIT":         2,
	"BL_ROLLBACK":       3,
	"BL_DML_DEPRECATED": 4,
	"BL_DDL":            5,
	"BL_SET":            6,
	"BL_INSERT":         7,
	"BL_UPDATE":         8,
	"BL_DELETE":         9,
}

func (x BinlogTransaction_Statement_Category) String() string {
	return proto.EnumName(BinlogTransaction_Statement_Category_name, int32(x))
}
func (BinlogTransaction_Statement_Category) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor0, []int{1, 0, 0}
}

// Charset is the per-statement charset info from a QUERY_EVENT binlog entry.
type Charset struct {
	// @@session.character_set_client
	Client int32 `protobuf:"varint,1,opt,name=client" json:"client,omitempty"`
	// @@session.collation_connection
	Conn int32 `protobuf:"varint,2,opt,name=conn" json:"conn,omitempty"`
	// @@session.collation_server
	Server int32 `protobuf:"varint,3,opt,name=server" json:"server,omitempty"`
}

func (m *Charset) Reset()                    { *m = Charset{} }
func (m *Charset) String() string            { return proto.CompactTextString(m) }
func (*Charset) ProtoMessage()               {}
func (*Charset) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// BinlogTransaction describes a transaction inside the binlogs.
// It is streamed by vttablet for filtered replication, used during resharding.
type BinlogTransaction struct {
	// the statements in this transaction
	Statements []*BinlogTransaction_Statement `protobuf:"bytes,1,rep,name=statements" json:"statements,omitempty"`
	// The Event Token for this event.
	EventToken *query.EventToken `protobuf:"bytes,4,opt,name=event_token,json=eventToken" json:"event_token,omitempty"`
}

func (m *BinlogTransaction) Reset()                    { *m = BinlogTransaction{} }
func (m *BinlogTransaction) String() string            { return proto.CompactTextString(m) }
func (*BinlogTransaction) ProtoMessage()               {}
func (*BinlogTransaction) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *BinlogTransaction) GetStatements() []*BinlogTransaction_Statement {
	if m != nil {
		return m.Statements
	}
	return nil
}

func (m *BinlogTransaction) GetEventToken() *query.EventToken {
	if m != nil {
		return m.EventToken
	}
	return nil
}

type BinlogTransaction_Statement struct {
	// what type of statement is this?
	Category BinlogTransaction_Statement_Category `protobuf:"varint,1,opt,name=category,enum=binlogdata.BinlogTransaction_Statement_Category" json:"category,omitempty"`
	// charset of this statement, if different from pre-negotiated default.
	Charset *Charset `protobuf:"bytes,2,opt,name=charset" json:"charset,omitempty"`
	// the sql
	Sql []byte `protobuf:"bytes,3,opt,name=sql,proto3" json:"sql,omitempty"`
}

func (m *BinlogTransaction_Statement) Reset()                    { *m = BinlogTransaction_Statement{} }
func (m *BinlogTransaction_Statement) String() string            { return proto.CompactTextString(m) }
func (*BinlogTransaction_Statement) ProtoMessage()               {}
func (*BinlogTransaction_Statement) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1, 0} }

func (m *BinlogTransaction_Statement) GetCharset() *Charset {
	if m != nil {
		return m.Charset
	}
	return nil
}

// StreamKeyRangeRequest is the payload to StreamKeyRange
type StreamKeyRangeRequest struct {
	// where to start
	Position string `protobuf:"bytes,1,opt,name=position" json:"position,omitempty"`
	// what to get
	KeyRange *topodata.KeyRange `protobuf:"bytes,2,opt,name=key_range,json=keyRange" json:"key_range,omitempty"`
	// default charset on the player side
	Charset *Charset `protobuf:"bytes,3,opt,name=charset" json:"charset,omitempty"`
}

func (m *StreamKeyRangeRequest) Reset()                    { *m = StreamKeyRangeRequest{} }
func (m *StreamKeyRangeRequest) String() string            { return proto.CompactTextString(m) }
func (*StreamKeyRangeRequest) ProtoMessage()               {}
func (*StreamKeyRangeRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *StreamKeyRangeRequest) GetKeyRange() *topodata.KeyRange {
	if m != nil {
		return m.KeyRange
	}
	return nil
}

func (m *StreamKeyRangeRequest) GetCharset() *Charset {
	if m != nil {
		return m.Charset
	}
	return nil
}

// StreamKeyRangeResponse is the response from StreamKeyRange
type StreamKeyRangeResponse struct {
	BinlogTransaction *BinlogTransaction `protobuf:"bytes,1,opt,name=binlog_transaction,json=binlogTransaction" json:"binlog_transaction,omitempty"`
}

func (m *StreamKeyRangeResponse) Reset()                    { *m = StreamKeyRangeResponse{} }
func (m *StreamKeyRangeResponse) String() string            { return proto.CompactTextString(m) }
func (*StreamKeyRangeResponse) ProtoMessage()               {}
func (*StreamKeyRangeResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *StreamKeyRangeResponse) GetBinlogTransaction() *BinlogTransaction {
	if m != nil {
		return m.BinlogTransaction
	}
	return nil
}

// StreamTablesRequest is the payload to StreamTables
type StreamTablesRequest struct {
	// where to start
	Position string `protobuf:"bytes,1,opt,name=position" json:"position,omitempty"`
	// what to get
	Tables []string `protobuf:"bytes,2,rep,name=tables" json:"tables,omitempty"`
	// default charset on the player side
	Charset *Charset `protobuf:"bytes,3,opt,name=charset" json:"charset,omitempty"`
}

func (m *StreamTablesRequest) Reset()                    { *m = StreamTablesRequest{} }
func (m *StreamTablesRequest) String() string            { return proto.CompactTextString(m) }
func (*StreamTablesRequest) ProtoMessage()               {}
func (*StreamTablesRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *StreamTablesRequest) GetCharset() *Charset {
	if m != nil {
		return m.Charset
	}
	return nil
}

// StreamTablesResponse is the response from StreamTables
type StreamTablesResponse struct {
	BinlogTransaction *BinlogTransaction `protobuf:"bytes,1,opt,name=binlog_transaction,json=binlogTransaction" json:"binlog_transaction,omitempty"`
}

func (m *StreamTablesResponse) Reset()                    { *m = StreamTablesResponse{} }
func (m *StreamTablesResponse) String() string            { return proto.CompactTextString(m) }
func (*StreamTablesResponse) ProtoMessage()               {}
func (*StreamTablesResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *StreamTablesResponse) GetBinlogTransaction() *BinlogTransaction {
	if m != nil {
		return m.BinlogTransaction
	}
	return nil
}

func init() {
	proto.RegisterType((*Charset)(nil), "binlogdata.Charset")
	proto.RegisterType((*BinlogTransaction)(nil), "binlogdata.BinlogTransaction")
	proto.RegisterType((*BinlogTransaction_Statement)(nil), "binlogdata.BinlogTransaction.Statement")
	proto.RegisterType((*StreamKeyRangeRequest)(nil), "binlogdata.StreamKeyRangeRequest")
	proto.RegisterType((*StreamKeyRangeResponse)(nil), "binlogdata.StreamKeyRangeResponse")
	proto.RegisterType((*StreamTablesRequest)(nil), "binlogdata.StreamTablesRequest")
	proto.RegisterType((*StreamTablesResponse)(nil), "binlogdata.StreamTablesResponse")
	proto.RegisterEnum("binlogdata.BinlogTransaction_Statement_Category", BinlogTransaction_Statement_Category_name, BinlogTransaction_Statement_Category_value)
}

func init() { proto.RegisterFile("binlogdata.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 540 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x54, 0x5d, 0x6e, 0xda, 0x40,
	0x10, 0xae, 0xb1, 0x43, 0xec, 0x71, 0x9a, 0x2c, 0x9b, 0x26, 0xb2, 0x90, 0x2a, 0x21, 0xbf, 0x94,
	0x97, 0xba, 0x95, 0x7b, 0x02, 0x6c, 0xaf, 0x10, 0xc9, 0x02, 0xd1, 0xe2, 0xbc, 0xf4, 0xc5, 0x32,
	0x64, 0x4b, 0x11, 0xc4, 0x06, 0xef, 0x26, 0x2a, 0xe7, 0xe8, 0x29, 0x7a, 0x91, 0xde, 0xa4, 0xf7,
	0xa8, 0xfc, 0x83, 0xa1, 0xa9, 0xd4, 0xa6, 0x0f, 0x7d, 0x9b, 0x6f, 0xe6, 0x9b, 0x6f, 0x66, 0x3e,
	0xaf, 0x0c, 0x68, 0xba, 0x48, 0x56, 0xe9, 0xfc, 0x2e, 0x96, 0xb1, 0xb3, 0xce, 0x52, 0x99, 0x62,
	0xd8, 0x67, 0xda, 0xe6, 0xe6, 0x81, 0x67, 0xdb, 0xb2, 0xd0, 0x3e, 0x95, 0xe9, 0x3a, 0xdd, 0x13,
	0xed, 0x21, 0x1c, 0xfb, 0x9f, 0xe3, 0x4c, 0x70, 0x89, 0x2f, 0xa1, 0x39, 0x5b, 0x2d, 0x78, 0x22,
	0x2d, 0xa5, 0xa3, 0x74, 0x8f, 0x58, 0x85, 0x30, 0x06, 0x6d, 0x96, 0x26, 0x89, 0xd5, 0x28, 0xb2,
	0x45, 0x9c, 0x73, 0x05, 0xcf, 0x1e, 0x79, 0x66, 0xa9, 0x25, 0xb7, 0x44, 0xf6, 0x0f, 0x15, 0x5a,
	0x5e, 0x31, 0x3a, 0xcc, 0xe2, 0x44, 0xc4, 0x33, 0xb9, 0x48, 0x13, 0xdc, 0x07, 0x10, 0x32, 0x96,
	0xfc, 0x9e, 0x27, 0x52, 0x58, 0x4a, 0x47, 0xed, 0x9a, 0xee, 0x1b, 0xe7, 0x60, 0xe9, 0xdf, 0x5a,
	0x9c, 0xc9, 0x8e, 0xcf, 0x0e, 0x5a, 0xb1, 0x0b, 0x26, 0x7f, 0xe4, 0x89, 0x8c, 0x64, 0xba, 0xe4,
	0x89, 0xa5, 0x75, 0x94, 0xae, 0xe9, 0xb6, 0x9c, 0xf2, 0x40, 0x92, 0x57, 0xc2, 0xbc, 0xc0, 0x80,
	0xd7, 0x71, 0xfb, 0x7b, 0x03, 0x8c, 0x5a, 0x0d, 0x53, 0xd0, 0x67, 0xb1, 0xe4, 0xf3, 0x34, 0xdb,
	0x16, 0x67, 0x9e, 0xba, 0xef, 0x9f, 0xb9, 0x88, 0xe3, 0x57, 0x7d, 0xac, 0x56, 0xc0, 0x6f, 0xe1,
	0x78, 0x56, 0xba, 0x57, 0xb8, 0x63, 0xba, 0xe7, 0x87, 0x62, 0x95, 0xb1, 0x6c, 0xc7, 0xc1, 0x08,
	0x54, 0xb1, 0x59, 0x15, 0x96, 0x9d, 0xb0, 0x3c, 0xb4, 0xbf, 0x29, 0xa0, 0xef, 0x74, 0xf1, 0x39,
	0x9c, 0x79, 0x34, 0xba, 0x1d, 0x31, 0xe2, 0x8f, 0xfb, 0xa3, 0xc1, 0x47, 0x12, 0xa0, 0x17, 0xf8,
	0x04, 0x74, 0x8f, 0x46, 0x1e, 0xe9, 0x0f, 0x46, 0x48, 0xc1, 0x2f, 0xc1, 0xf0, 0x68, 0xe4, 0x8f,
	0x87, 0xc3, 0x41, 0x88, 0x1a, 0xf8, 0x0c, 0x4c, 0x8f, 0x46, 0x6c, 0x4c, 0xa9, 0xd7, 0xf3, 0xaf,
	0x91, 0x8a, 0x2f, 0xa0, 0xe5, 0xd1, 0x28, 0x18, 0xd2, 0x28, 0x20, 0x37, 0x8c, 0xf8, 0xbd, 0x90,
	0x04, 0x48, 0xc3, 0x00, 0xcd, 0x3c, 0x1d, 0x50, 0x74, 0x54, 0xc5, 0x13, 0x12, 0xa2, 0x66, 0x25,
	0x37, 0x18, 0x4d, 0x08, 0x0b, 0xd1, 0x71, 0x05, 0x6f, 0x6f, 0x82, 0x5e, 0x48, 0x90, 0x5e, 0xc1,
	0x80, 0x50, 0x12, 0x12, 0x64, 0x5c, 0x69, 0x7a, 0x03, 0xa9, 0x57, 0x9a, 0xae, 0x22, 0xcd, 0xfe,
	0xaa, 0xc0, 0xc5, 0x44, 0x66, 0x3c, 0xbe, 0xbf, 0xe6, 0x5b, 0x16, 0x27, 0x73, 0xce, 0xf8, 0xe6,
	0x81, 0x0b, 0x89, 0xdb, 0xa0, 0xaf, 0x53, 0xb1, 0xc8, 0xbd, 0x2b, 0x0c, 0x36, 0x58, 0x8d, 0xf1,
	0x3b, 0x30, 0x96, 0x7c, 0x1b, 0x65, 0x39, 0xbf, 0x32, 0x0c, 0x3b, 0xf5, 0x83, 0xac, 0x95, 0xf4,
	0x65, 0x15, 0x1d, 0xfa, 0xab, 0xfe, 0xdd, 0x5f, 0xfb, 0x13, 0x5c, 0x3e, 0x5d, 0x4a, 0xac, 0xd3,
	0x44, 0x70, 0x4c, 0x01, 0x97, 0x8d, 0x91, 0xdc, 0x7f, 0xdb, 0x62, 0x3f, 0xd3, 0x7d, 0xfd, 0xc7,
	0x07, 0xc0, 0x5a, 0xd3, 0xa7, 0x29, 0xfb, 0x0b, 0x9c, 0x97, 0x73, 0xc2, 0x78, 0xba, 0xe2, 0xe2,
	0x39, 0xa7, 0x5f, 0x42, 0x53, 0x16, 0x64, 0xab, 0xd1, 0x51, 0xbb, 0x06, 0xab, 0xd0, 0xbf, 0x5e,
	0x78, 0x07, 0xaf, 0x7e, 0x9d, 0xfc, 0x3f, 0xee, 0x9b, 0x36, 0x8b, 0x7f, 0xc3, 0x87, 0x9f, 0x01,
	0x00, 0x00, 0xff, 0xff, 0xf4, 0x20, 0x24, 0xf0, 0x58, 0x04, 0x00, 0x00,
}
