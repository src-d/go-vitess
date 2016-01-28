// Code generated by protoc-gen-go.
// source: replicationdata.proto
// DO NOT EDIT!

/*
Package replicationdata is a generated protocol buffer package.

It is generated from these files:
	replicationdata.proto

It has these top-level messages:
	Status
*/
package replicationdata

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.ProtoPackageIsVersion1

// Status is the replication status for MySQL (returned by 'show slave status'
// and parsed into a Position and fields).
type Status struct {
	Position            string `protobuf:"bytes,1,opt,name=position" json:"position,omitempty"`
	SlaveIoRunning      bool   `protobuf:"varint,2,opt,name=slave_io_running" json:"slave_io_running,omitempty"`
	SlaveSqlRunning     bool   `protobuf:"varint,3,opt,name=slave_sql_running" json:"slave_sql_running,omitempty"`
	SecondsBehindMaster uint32 `protobuf:"varint,4,opt,name=seconds_behind_master" json:"seconds_behind_master,omitempty"`
	MasterHost          string `protobuf:"bytes,5,opt,name=master_host" json:"master_host,omitempty"`
	MasterPort          int32  `protobuf:"varint,6,opt,name=master_port" json:"master_port,omitempty"`
	MasterConnectRetry  int32  `protobuf:"varint,7,opt,name=master_connect_retry" json:"master_connect_retry,omitempty"`
}

func (m *Status) Reset()                    { *m = Status{} }
func (m *Status) String() string            { return proto.CompactTextString(m) }
func (*Status) ProtoMessage()               {}
func (*Status) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func init() {
	proto.RegisterType((*Status)(nil), "replicationdata.Status")
}

var fileDescriptor0 = []byte{
	// 190 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x5c, 0xcf, 0x41, 0xaa, 0xc2, 0x30,
	0x10, 0xc6, 0x71, 0xfa, 0x9e, 0xad, 0x75, 0x44, 0xac, 0xd1, 0x42, 0x04, 0x05, 0x71, 0xe5, 0xca,
	0x8d, 0x47, 0xf1, 0x00, 0x21, 0x6d, 0x83, 0x0d, 0xd4, 0x24, 0x66, 0xa6, 0x82, 0x17, 0xf3, 0x7c,
	0xa6, 0xad, 0x28, 0xb8, 0xcc, 0xff, 0x07, 0x5f, 0x18, 0xc8, 0xbd, 0x72, 0x8d, 0x2e, 0x25, 0x69,
	0x6b, 0x2a, 0x49, 0xf2, 0xe8, 0xbc, 0x25, 0xcb, 0xe6, 0x3f, 0x79, 0xff, 0x8c, 0x20, 0x39, 0x93,
	0xa4, 0x16, 0x59, 0x06, 0xa9, 0xb3, 0xa8, 0x3b, 0xe2, 0xd1, 0x2e, 0x3a, 0x4c, 0x18, 0x87, 0x0c,
	0x1b, 0x79, 0x57, 0x42, 0x5b, 0xe1, 0x5b, 0x63, 0xb4, 0xb9, 0xf0, 0xbf, 0x20, 0x29, 0x5b, 0xc3,
	0x62, 0x10, 0xbc, 0x35, 0x1f, 0xfa, 0xef, 0x69, 0x0b, 0x39, 0xaa, 0x32, 0xcc, 0xa3, 0x28, 0x54,
	0xad, 0x4d, 0x25, 0xae, 0x12, 0x49, 0x79, 0x3e, 0x0a, 0x3c, 0x63, 0x4b, 0x98, 0x0e, 0x6f, 0x51,
	0x5b, 0x24, 0x1e, 0xf7, 0x1f, 0x7d, 0xa3, 0xb3, 0x9e, 0x78, 0x12, 0x62, 0xcc, 0x36, 0xb0, 0x7a,
	0xc7, 0xb0, 0x66, 0x54, 0x49, 0xc2, 0x2b, 0xf2, 0x0f, 0x3e, 0xee, 0xb4, 0x48, 0xfa, 0x83, 0x4e,
	0xaf, 0x00, 0x00, 0x00, 0xff, 0xff, 0xa4, 0x78, 0x89, 0xe0, 0xe9, 0x00, 0x00, 0x00,
}