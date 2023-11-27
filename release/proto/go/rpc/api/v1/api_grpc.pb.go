// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: rpc/api/v1/api.proto

package apiv1

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

const (
	PoetService_PowParams_FullMethodName = "/rpc.api.v1.PoetService/PowParams"
	PoetService_Submit_FullMethodName    = "/rpc.api.v1.PoetService/Submit"
	PoetService_Info_FullMethodName      = "/rpc.api.v1.PoetService/Info"
	PoetService_Proof_FullMethodName     = "/rpc.api.v1.PoetService/Proof"
)

// PoetServiceClient is the client API for PoetService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PoetServiceClient interface {
	// Deprecated: Do not use.
	PowParams(ctx context.Context, in *PowParamsRequest, opts ...grpc.CallOption) (*PowParamsResponse, error)
	// *
	// Submit registers data to the service's current open round,
	// to be included its later generated proof.
	Submit(ctx context.Context, in *SubmitRequest, opts ...grpc.CallOption) (*SubmitResponse, error)
	// *
	// Info returns general information concerning the service,
	// including its identity pubkey.
	Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error)
	// *
	// roof returns the generated proof for given round id.
	Proof(ctx context.Context, in *ProofRequest, opts ...grpc.CallOption) (*ProofResponse, error)
}

type poetServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPoetServiceClient(cc grpc.ClientConnInterface) PoetServiceClient {
	return &poetServiceClient{cc}
}

// Deprecated: Do not use.
func (c *poetServiceClient) PowParams(ctx context.Context, in *PowParamsRequest, opts ...grpc.CallOption) (*PowParamsResponse, error) {
	out := new(PowParamsResponse)
	err := c.cc.Invoke(ctx, PoetService_PowParams_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *poetServiceClient) Submit(ctx context.Context, in *SubmitRequest, opts ...grpc.CallOption) (*SubmitResponse, error) {
	out := new(SubmitResponse)
	err := c.cc.Invoke(ctx, PoetService_Submit_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *poetServiceClient) Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error) {
	out := new(InfoResponse)
	err := c.cc.Invoke(ctx, PoetService_Info_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *poetServiceClient) Proof(ctx context.Context, in *ProofRequest, opts ...grpc.CallOption) (*ProofResponse, error) {
	out := new(ProofResponse)
	err := c.cc.Invoke(ctx, PoetService_Proof_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PoetServiceServer is the server API for PoetService service.
// All implementations should embed UnimplementedPoetServiceServer
// for forward compatibility
type PoetServiceServer interface {
	// Deprecated: Do not use.
	PowParams(context.Context, *PowParamsRequest) (*PowParamsResponse, error)
	// *
	// Submit registers data to the service's current open round,
	// to be included its later generated proof.
	Submit(context.Context, *SubmitRequest) (*SubmitResponse, error)
	// *
	// Info returns general information concerning the service,
	// including its identity pubkey.
	Info(context.Context, *InfoRequest) (*InfoResponse, error)
	// *
	// roof returns the generated proof for given round id.
	Proof(context.Context, *ProofRequest) (*ProofResponse, error)
}

// UnimplementedPoetServiceServer should be embedded to have forward compatible implementations.
type UnimplementedPoetServiceServer struct {
}

func (UnimplementedPoetServiceServer) PowParams(context.Context, *PowParamsRequest) (*PowParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PowParams not implemented")
}
func (UnimplementedPoetServiceServer) Submit(context.Context, *SubmitRequest) (*SubmitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Submit not implemented")
}
func (UnimplementedPoetServiceServer) Info(context.Context, *InfoRequest) (*InfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Info not implemented")
}
func (UnimplementedPoetServiceServer) Proof(context.Context, *ProofRequest) (*ProofResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Proof not implemented")
}

// UnsafePoetServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PoetServiceServer will
// result in compilation errors.
type UnsafePoetServiceServer interface {
	mustEmbedUnimplementedPoetServiceServer()
}

func RegisterPoetServiceServer(s grpc.ServiceRegistrar, srv PoetServiceServer) {
	s.RegisterService(&PoetService_ServiceDesc, srv)
}

func _PoetService_PowParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PowParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PoetServiceServer).PowParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PoetService_PowParams_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PoetServiceServer).PowParams(ctx, req.(*PowParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PoetService_Submit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SubmitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PoetServiceServer).Submit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PoetService_Submit_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PoetServiceServer).Submit(ctx, req.(*SubmitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PoetService_Info_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PoetServiceServer).Info(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PoetService_Info_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PoetServiceServer).Info(ctx, req.(*InfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PoetService_Proof_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProofRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PoetServiceServer).Proof(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PoetService_Proof_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PoetServiceServer).Proof(ctx, req.(*ProofRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// PoetService_ServiceDesc is the grpc.ServiceDesc for PoetService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PoetService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "rpc.api.v1.PoetService",
	HandlerType: (*PoetServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PowParams",
			Handler:    _PoetService_PowParams_Handler,
		},
		{
			MethodName: "Submit",
			Handler:    _PoetService_Submit_Handler,
		},
		{
			MethodName: "Info",
			Handler:    _PoetService_Info_Handler,
		},
		{
			MethodName: "Proof",
			Handler:    _PoetService_Proof_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rpc/api/v1/api.proto",
}
