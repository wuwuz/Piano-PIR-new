// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: query/query.proto

package query

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// QueryServiceClient is the client API for QueryService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type QueryServiceClient interface {
	// rpc QueryDatabase (SimpleQuery) returns (QueryResponse) {}
	// rpc SingleQuery (singleQuery) returns (CuckooBucketResponse) {}
	PlaintextQuery(ctx context.Context, in *PlaintextQueryMsg, opts ...grpc.CallOption) (*PlaintextResponse, error)
	FullSetQuery(ctx context.Context, in *FullSetQueryMsg, opts ...grpc.CallOption) (*FullSetResponse, error)
	PunctSetQuery(ctx context.Context, in *PunctSetQueryMsg, opts ...grpc.CallOption) (*PunctSetResponse, error)
	// rpc BatchedFullSetQuery (stream FullSetQueryMsg) returns (stream FullSetResponse) {}
	BatchedFullSetQuery(ctx context.Context, in *BatchedFullSetQueryMsg, opts ...grpc.CallOption) (*BatchedFullSetResponse, error)
	FetchFullDB(ctx context.Context, in *FetchFullDBMsg, opts ...grpc.CallOption) (QueryService_FetchFullDBClient, error)
	SetParityQuery(ctx context.Context, in *SetParityQueryMsg, opts ...grpc.CallOption) (*SetParityQueryResponse, error)
}

type queryServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryServiceClient(cc grpc.ClientConnInterface) QueryServiceClient {
	return &queryServiceClient{cc}
}

func (c *queryServiceClient) PlaintextQuery(ctx context.Context, in *PlaintextQueryMsg, opts ...grpc.CallOption) (*PlaintextResponse, error) {
	out := new(PlaintextResponse)
	err := c.cc.Invoke(ctx, "/query.QueryService/PlaintextQuery", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryServiceClient) FullSetQuery(ctx context.Context, in *FullSetQueryMsg, opts ...grpc.CallOption) (*FullSetResponse, error) {
	out := new(FullSetResponse)
	err := c.cc.Invoke(ctx, "/query.QueryService/FullSetQuery", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryServiceClient) PunctSetQuery(ctx context.Context, in *PunctSetQueryMsg, opts ...grpc.CallOption) (*PunctSetResponse, error) {
	out := new(PunctSetResponse)
	err := c.cc.Invoke(ctx, "/query.QueryService/PunctSetQuery", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryServiceClient) BatchedFullSetQuery(ctx context.Context, in *BatchedFullSetQueryMsg, opts ...grpc.CallOption) (*BatchedFullSetResponse, error) {
	out := new(BatchedFullSetResponse)
	err := c.cc.Invoke(ctx, "/query.QueryService/BatchedFullSetQuery", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryServiceClient) FetchFullDB(ctx context.Context, in *FetchFullDBMsg, opts ...grpc.CallOption) (QueryService_FetchFullDBClient, error) {
	stream, err := c.cc.NewStream(ctx, &QueryService_ServiceDesc.Streams[0], "/query.QueryService/FetchFullDB", opts...)
	if err != nil {
		return nil, err
	}
	x := &queryServiceFetchFullDBClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type QueryService_FetchFullDBClient interface {
	Recv() (*DBChunk, error)
	grpc.ClientStream
}

type queryServiceFetchFullDBClient struct {
	grpc.ClientStream
}

func (x *queryServiceFetchFullDBClient) Recv() (*DBChunk, error) {
	m := new(DBChunk)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *queryServiceClient) SetParityQuery(ctx context.Context, in *SetParityQueryMsg, opts ...grpc.CallOption) (*SetParityQueryResponse, error) {
	out := new(SetParityQueryResponse)
	err := c.cc.Invoke(ctx, "/query.QueryService/SetParityQuery", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServiceServer is the server API for QueryService service.
// All implementations must embed UnimplementedQueryServiceServer
// for forward compatibility
type QueryServiceServer interface {
	// rpc QueryDatabase (SimpleQuery) returns (QueryResponse) {}
	// rpc SingleQuery (singleQuery) returns (CuckooBucketResponse) {}
	PlaintextQuery(context.Context, *PlaintextQueryMsg) (*PlaintextResponse, error)
	FullSetQuery(context.Context, *FullSetQueryMsg) (*FullSetResponse, error)
	PunctSetQuery(context.Context, *PunctSetQueryMsg) (*PunctSetResponse, error)
	// rpc BatchedFullSetQuery (stream FullSetQueryMsg) returns (stream FullSetResponse) {}
	BatchedFullSetQuery(context.Context, *BatchedFullSetQueryMsg) (*BatchedFullSetResponse, error)
	FetchFullDB(*FetchFullDBMsg, QueryService_FetchFullDBServer) error
	SetParityQuery(context.Context, *SetParityQueryMsg) (*SetParityQueryResponse, error)
	mustEmbedUnimplementedQueryServiceServer()
}

// UnimplementedQueryServiceServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServiceServer struct {
}

func (UnimplementedQueryServiceServer) PlaintextQuery(context.Context, *PlaintextQueryMsg) (*PlaintextResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PlaintextQuery not implemented")
}
func (UnimplementedQueryServiceServer) FullSetQuery(context.Context, *FullSetQueryMsg) (*FullSetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FullSetQuery not implemented")
}
func (UnimplementedQueryServiceServer) PunctSetQuery(context.Context, *PunctSetQueryMsg) (*PunctSetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PunctSetQuery not implemented")
}
func (UnimplementedQueryServiceServer) BatchedFullSetQuery(context.Context, *BatchedFullSetQueryMsg) (*BatchedFullSetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BatchedFullSetQuery not implemented")
}
func (UnimplementedQueryServiceServer) FetchFullDB(*FetchFullDBMsg, QueryService_FetchFullDBServer) error {
	return status.Errorf(codes.Unimplemented, "method FetchFullDB not implemented")
}
func (UnimplementedQueryServiceServer) SetParityQuery(context.Context, *SetParityQueryMsg) (*SetParityQueryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetParityQuery not implemented")
}
func (UnimplementedQueryServiceServer) mustEmbedUnimplementedQueryServiceServer() {}

// UnsafeQueryServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to QueryServiceServer will
// result in compilation errors.
type UnsafeQueryServiceServer interface {
	mustEmbedUnimplementedQueryServiceServer()
}

func RegisterQueryServiceServer(s grpc.ServiceRegistrar, srv QueryServiceServer) {
	s.RegisterService(&QueryService_ServiceDesc, srv)
}

func _QueryService_PlaintextQuery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PlaintextQueryMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServiceServer).PlaintextQuery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/query.QueryService/PlaintextQuery",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServiceServer).PlaintextQuery(ctx, req.(*PlaintextQueryMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _QueryService_FullSetQuery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FullSetQueryMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServiceServer).FullSetQuery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/query.QueryService/FullSetQuery",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServiceServer).FullSetQuery(ctx, req.(*FullSetQueryMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _QueryService_PunctSetQuery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PunctSetQueryMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServiceServer).PunctSetQuery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/query.QueryService/PunctSetQuery",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServiceServer).PunctSetQuery(ctx, req.(*PunctSetQueryMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _QueryService_BatchedFullSetQuery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BatchedFullSetQueryMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServiceServer).BatchedFullSetQuery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/query.QueryService/BatchedFullSetQuery",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServiceServer).BatchedFullSetQuery(ctx, req.(*BatchedFullSetQueryMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _QueryService_FetchFullDB_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(FetchFullDBMsg)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(QueryServiceServer).FetchFullDB(m, &queryServiceFetchFullDBServer{stream})
}

type QueryService_FetchFullDBServer interface {
	Send(*DBChunk) error
	grpc.ServerStream
}

type queryServiceFetchFullDBServer struct {
	grpc.ServerStream
}

func (x *queryServiceFetchFullDBServer) Send(m *DBChunk) error {
	return x.ServerStream.SendMsg(m)
}

func _QueryService_SetParityQuery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetParityQueryMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServiceServer).SetParityQuery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/query.QueryService/SetParityQuery",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServiceServer).SetParityQuery(ctx, req.(*SetParityQueryMsg))
	}
	return interceptor(ctx, in, info, handler)
}

// QueryService_ServiceDesc is the grpc.ServiceDesc for QueryService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var QueryService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "query.QueryService",
	HandlerType: (*QueryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PlaintextQuery",
			Handler:    _QueryService_PlaintextQuery_Handler,
		},
		{
			MethodName: "FullSetQuery",
			Handler:    _QueryService_FullSetQuery_Handler,
		},
		{
			MethodName: "PunctSetQuery",
			Handler:    _QueryService_PunctSetQuery_Handler,
		},
		{
			MethodName: "BatchedFullSetQuery",
			Handler:    _QueryService_BatchedFullSetQuery_Handler,
		},
		{
			MethodName: "SetParityQuery",
			Handler:    _QueryService_SetParityQuery_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "FetchFullDB",
			Handler:       _QueryService_FetchFullDB_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "query/query.proto",
}
