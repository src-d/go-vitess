// Code generated by protoc-gen-go.
// source: vtworkerservice.proto
// DO NOT EDIT!

/*
Package vtworkerservice is a generated protocol buffer package.

It is generated from these files:
	vtworkerservice.proto

It has these top-level messages:
*/
package vtworkerservice

import proto "github.com/golang/protobuf/proto"
import vtworkerdata "github.com/youtube/vitess/go/vt/proto/vtworkerdata"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal

// Client API for Vtworker service

type VtworkerClient interface {
	// ExecuteVtworkerCommand allows to run a vtworker command by specifying the
	// same arguments as on the command line.
	ExecuteVtworkerCommand(ctx context.Context, in *vtworkerdata.ExecuteVtworkerCommandRequest, opts ...grpc.CallOption) (Vtworker_ExecuteVtworkerCommandClient, error)
}

type vtworkerClient struct {
	cc *grpc.ClientConn
}

func NewVtworkerClient(cc *grpc.ClientConn) VtworkerClient {
	return &vtworkerClient{cc}
}

func (c *vtworkerClient) ExecuteVtworkerCommand(ctx context.Context, in *vtworkerdata.ExecuteVtworkerCommandRequest, opts ...grpc.CallOption) (Vtworker_ExecuteVtworkerCommandClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_Vtworker_serviceDesc.Streams[0], c.cc, "/vtworkerservice.Vtworker/ExecuteVtworkerCommand", opts...)
	if err != nil {
		return nil, err
	}
	x := &vtworkerExecuteVtworkerCommandClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Vtworker_ExecuteVtworkerCommandClient interface {
	Recv() (*vtworkerdata.ExecuteVtworkerCommandResponse, error)
	grpc.ClientStream
}

type vtworkerExecuteVtworkerCommandClient struct {
	grpc.ClientStream
}

func (x *vtworkerExecuteVtworkerCommandClient) Recv() (*vtworkerdata.ExecuteVtworkerCommandResponse, error) {
	m := new(vtworkerdata.ExecuteVtworkerCommandResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Server API for Vtworker service

type VtworkerServer interface {
	// ExecuteVtworkerCommand allows to run a vtworker command by specifying the
	// same arguments as on the command line.
	ExecuteVtworkerCommand(*vtworkerdata.ExecuteVtworkerCommandRequest, Vtworker_ExecuteVtworkerCommandServer) error
}

func RegisterVtworkerServer(s *grpc.Server, srv VtworkerServer) {
	s.RegisterService(&_Vtworker_serviceDesc, srv)
}

func _Vtworker_ExecuteVtworkerCommand_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(vtworkerdata.ExecuteVtworkerCommandRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(VtworkerServer).ExecuteVtworkerCommand(m, &vtworkerExecuteVtworkerCommandServer{stream})
}

type Vtworker_ExecuteVtworkerCommandServer interface {
	Send(*vtworkerdata.ExecuteVtworkerCommandResponse) error
	grpc.ServerStream
}

type vtworkerExecuteVtworkerCommandServer struct {
	grpc.ServerStream
}

func (x *vtworkerExecuteVtworkerCommandServer) Send(m *vtworkerdata.ExecuteVtworkerCommandResponse) error {
	return x.ServerStream.SendMsg(m)
}

var _Vtworker_serviceDesc = grpc.ServiceDesc{
	ServiceName: "vtworkerservice.Vtworker",
	HandlerType: (*VtworkerServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ExecuteVtworkerCommand",
			Handler:       _Vtworker_ExecuteVtworkerCommand_Handler,
			ServerStreams: true,
		},
	},
}
