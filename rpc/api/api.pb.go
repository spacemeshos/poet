// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api.proto

package api

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "google.golang.org/genproto/googleapis/api/annotations"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type SubmitRequest struct {
	Challenge            []byte   `protobuf:"bytes,1,opt,name=challenge,proto3" json:"challenge,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SubmitRequest) Reset()         { *m = SubmitRequest{} }
func (m *SubmitRequest) String() string { return proto.CompactTextString(m) }
func (*SubmitRequest) ProtoMessage()    {}
func (*SubmitRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_1adc4fc7c9bb1fa2, []int{0}
}
func (m *SubmitRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SubmitRequest.Unmarshal(m, b)
}
func (m *SubmitRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SubmitRequest.Marshal(b, m, deterministic)
}
func (dst *SubmitRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SubmitRequest.Merge(dst, src)
}
func (m *SubmitRequest) XXX_Size() int {
	return xxx_messageInfo_SubmitRequest.Size(m)
}
func (m *SubmitRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_SubmitRequest.DiscardUnknown(m)
}

var xxx_messageInfo_SubmitRequest proto.InternalMessageInfo

func (m *SubmitRequest) GetChallenge() []byte {
	if m != nil {
		return m.Challenge
	}
	return nil
}

type SubmitResponse struct {
	RoundId              int32    `protobuf:"varint,1,opt,name=roundId,proto3" json:"roundId,omitempty"`
	PoetId               []byte   `protobuf:"bytes,2,opt,name=poetId,proto3" json:"poetId,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SubmitResponse) Reset()         { *m = SubmitResponse{} }
func (m *SubmitResponse) String() string { return proto.CompactTextString(m) }
func (*SubmitResponse) ProtoMessage()    {}
func (*SubmitResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_1adc4fc7c9bb1fa2, []int{1}
}
func (m *SubmitResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SubmitResponse.Unmarshal(m, b)
}
func (m *SubmitResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SubmitResponse.Marshal(b, m, deterministic)
}
func (dst *SubmitResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SubmitResponse.Merge(dst, src)
}
func (m *SubmitResponse) XXX_Size() int {
	return xxx_messageInfo_SubmitResponse.Size(m)
}
func (m *SubmitResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_SubmitResponse.DiscardUnknown(m)
}

var xxx_messageInfo_SubmitResponse proto.InternalMessageInfo

func (m *SubmitResponse) GetRoundId() int32 {
	if m != nil {
		return m.RoundId
	}
	return 0
}

func (m *SubmitResponse) GetPoetId() []byte {
	if m != nil {
		return m.PoetId
	}
	return nil
}

type GetInfoRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetInfoRequest) Reset()         { *m = GetInfoRequest{} }
func (m *GetInfoRequest) String() string { return proto.CompactTextString(m) }
func (*GetInfoRequest) ProtoMessage()    {}
func (*GetInfoRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_1adc4fc7c9bb1fa2, []int{2}
}
func (m *GetInfoRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetInfoRequest.Unmarshal(m, b)
}
func (m *GetInfoRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetInfoRequest.Marshal(b, m, deterministic)
}
func (dst *GetInfoRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetInfoRequest.Merge(dst, src)
}
func (m *GetInfoRequest) XXX_Size() int {
	return xxx_messageInfo_GetInfoRequest.Size(m)
}
func (m *GetInfoRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetInfoRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetInfoRequest proto.InternalMessageInfo

type GetInfoResponse struct {
	OpenRoundId          int32    `protobuf:"varint,1,opt,name=openRoundId,proto3" json:"openRoundId,omitempty"`
	ExecutingRoundsIds   []int32  `protobuf:"varint,2,rep,packed,name=executingRoundsIds,proto3" json:"executingRoundsIds,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetInfoResponse) Reset()         { *m = GetInfoResponse{} }
func (m *GetInfoResponse) String() string { return proto.CompactTextString(m) }
func (*GetInfoResponse) ProtoMessage()    {}
func (*GetInfoResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_1adc4fc7c9bb1fa2, []int{3}
}
func (m *GetInfoResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetInfoResponse.Unmarshal(m, b)
}
func (m *GetInfoResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetInfoResponse.Marshal(b, m, deterministic)
}
func (dst *GetInfoResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetInfoResponse.Merge(dst, src)
}
func (m *GetInfoResponse) XXX_Size() int {
	return xxx_messageInfo_GetInfoResponse.Size(m)
}
func (m *GetInfoResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetInfoResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetInfoResponse proto.InternalMessageInfo

func (m *GetInfoResponse) GetOpenRoundId() int32 {
	if m != nil {
		return m.OpenRoundId
	}
	return 0
}

func (m *GetInfoResponse) GetExecutingRoundsIds() []int32 {
	if m != nil {
		return m.ExecutingRoundsIds
	}
	return nil
}

type MembershipProof struct {
	Index                int32    `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"`
	Root                 []byte   `protobuf:"bytes,2,opt,name=root,proto3" json:"root,omitempty"`
	Proof                [][]byte `protobuf:"bytes,3,rep,name=proof,proto3" json:"proof,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MembershipProof) Reset()         { *m = MembershipProof{} }
func (m *MembershipProof) String() string { return proto.CompactTextString(m) }
func (*MembershipProof) ProtoMessage()    {}
func (*MembershipProof) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_1adc4fc7c9bb1fa2, []int{4}
}
func (m *MembershipProof) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MembershipProof.Unmarshal(m, b)
}
func (m *MembershipProof) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MembershipProof.Marshal(b, m, deterministic)
}
func (dst *MembershipProof) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MembershipProof.Merge(dst, src)
}
func (m *MembershipProof) XXX_Size() int {
	return xxx_messageInfo_MembershipProof.Size(m)
}
func (m *MembershipProof) XXX_DiscardUnknown() {
	xxx_messageInfo_MembershipProof.DiscardUnknown(m)
}

var xxx_messageInfo_MembershipProof proto.InternalMessageInfo

func (m *MembershipProof) GetIndex() int32 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *MembershipProof) GetRoot() []byte {
	if m != nil {
		return m.Root
	}
	return nil
}

func (m *MembershipProof) GetProof() [][]byte {
	if m != nil {
		return m.Proof
	}
	return nil
}

type PoetProof struct {
	Phi                  []byte   `protobuf:"bytes,1,opt,name=phi,proto3" json:"phi,omitempty"`
	ProvenLeaves         [][]byte `protobuf:"bytes,2,rep,name=provenLeaves,proto3" json:"provenLeaves,omitempty"`
	ProofNodes           [][]byte `protobuf:"bytes,3,rep,name=proofNodes,proto3" json:"proofNodes,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PoetProof) Reset()         { *m = PoetProof{} }
func (m *PoetProof) String() string { return proto.CompactTextString(m) }
func (*PoetProof) ProtoMessage()    {}
func (*PoetProof) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_1adc4fc7c9bb1fa2, []int{5}
}
func (m *PoetProof) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PoetProof.Unmarshal(m, b)
}
func (m *PoetProof) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PoetProof.Marshal(b, m, deterministic)
}
func (dst *PoetProof) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PoetProof.Merge(dst, src)
}
func (m *PoetProof) XXX_Size() int {
	return xxx_messageInfo_PoetProof.Size(m)
}
func (m *PoetProof) XXX_DiscardUnknown() {
	xxx_messageInfo_PoetProof.DiscardUnknown(m)
}

var xxx_messageInfo_PoetProof proto.InternalMessageInfo

func (m *PoetProof) GetPhi() []byte {
	if m != nil {
		return m.Phi
	}
	return nil
}

func (m *PoetProof) GetProvenLeaves() [][]byte {
	if m != nil {
		return m.ProvenLeaves
	}
	return nil
}

func (m *PoetProof) GetProofNodes() [][]byte {
	if m != nil {
		return m.ProofNodes
	}
	return nil
}

func init() {
	proto.RegisterType((*SubmitRequest)(nil), "api.SubmitRequest")
	proto.RegisterType((*SubmitResponse)(nil), "api.SubmitResponse")
	proto.RegisterType((*GetInfoRequest)(nil), "api.GetInfoRequest")
	proto.RegisterType((*GetInfoResponse)(nil), "api.GetInfoResponse")
	proto.RegisterType((*MembershipProof)(nil), "api.MembershipProof")
	proto.RegisterType((*PoetProof)(nil), "api.PoetProof")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// PoetClient is the client API for Poet service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type PoetClient interface {
	Submit(ctx context.Context, in *SubmitRequest, opts ...grpc.CallOption) (*SubmitResponse, error)
	GetInfo(ctx context.Context, in *GetInfoRequest, opts ...grpc.CallOption) (*GetInfoResponse, error)
}

type poetClient struct {
	cc *grpc.ClientConn
}

func NewPoetClient(cc *grpc.ClientConn) PoetClient {
	return &poetClient{cc}
}

func (c *poetClient) Submit(ctx context.Context, in *SubmitRequest, opts ...grpc.CallOption) (*SubmitResponse, error) {
	out := new(SubmitResponse)
	err := c.cc.Invoke(ctx, "/api.Poet/Submit", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *poetClient) GetInfo(ctx context.Context, in *GetInfoRequest, opts ...grpc.CallOption) (*GetInfoResponse, error) {
	out := new(GetInfoResponse)
	err := c.cc.Invoke(ctx, "/api.Poet/GetInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PoetServer is the server API for Poet service.
type PoetServer interface {
	Submit(context.Context, *SubmitRequest) (*SubmitResponse, error)
	GetInfo(context.Context, *GetInfoRequest) (*GetInfoResponse, error)
}

func RegisterPoetServer(s *grpc.Server, srv PoetServer) {
	s.RegisterService(&_Poet_serviceDesc, srv)
}

func _Poet_Submit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SubmitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PoetServer).Submit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.Poet/Submit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PoetServer).Submit(ctx, req.(*SubmitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Poet_GetInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PoetServer).GetInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.Poet/GetInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PoetServer).GetInfo(ctx, req.(*GetInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Poet_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.Poet",
	HandlerType: (*PoetServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Submit",
			Handler:    _Poet_Submit_Handler,
		},
		{
			MethodName: "GetInfo",
			Handler:    _Poet_GetInfo_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api.proto",
}

func init() { proto.RegisterFile("api.proto", fileDescriptor_api_1adc4fc7c9bb1fa2) }

var fileDescriptor_api_1adc4fc7c9bb1fa2 = []byte{
	// 377 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x92, 0xc1, 0x8e, 0xd3, 0x30,
	0x10, 0x86, 0x95, 0x66, 0xdb, 0xa5, 0x43, 0xd8, 0x56, 0xc3, 0x82, 0xa2, 0x6a, 0x85, 0x22, 0x9f,
	0x2a, 0x24, 0x1a, 0x01, 0x37, 0x8e, 0x1c, 0x80, 0x48, 0x80, 0x96, 0xf0, 0x04, 0x6e, 0x33, 0x4d,
	0x2d, 0x75, 0x3d, 0x26, 0x76, 0xaa, 0x3d, 0xf3, 0x0a, 0x5c, 0x78, 0x2f, 0x5e, 0x81, 0x07, 0x41,
	0xb1, 0xbd, 0xb0, 0x91, 0xb8, 0x79, 0x3e, 0x8f, 0x3f, 0x5b, 0xff, 0x18, 0xe6, 0xd2, 0xa8, 0x8d,
	0xe9, 0xd8, 0x31, 0xa6, 0xd2, 0xa8, 0xd5, 0x55, 0xcb, 0xdc, 0x1e, 0xa9, 0x94, 0x46, 0x95, 0x52,
	0x6b, 0x76, 0xd2, 0x29, 0xd6, 0x36, 0xb4, 0x88, 0x17, 0xf0, 0xe8, 0x6b, 0xbf, 0xbd, 0x51, 0xae,
	0xa6, 0x6f, 0x3d, 0x59, 0x87, 0x57, 0x30, 0xdf, 0x1d, 0xe4, 0xf1, 0x48, 0xba, 0xa5, 0x3c, 0x29,
	0x92, 0x75, 0x56, 0xff, 0x03, 0xe2, 0x2d, 0x5c, 0xdc, 0xb5, 0x5b, 0xc3, 0xda, 0x12, 0xe6, 0x70,
	0xde, 0x71, 0xaf, 0x9b, 0xaa, 0xf1, 0xdd, 0xd3, 0xfa, 0xae, 0xc4, 0xa7, 0x30, 0x33, 0x4c, 0xae,
	0x6a, 0xf2, 0x89, 0xd7, 0xc4, 0x4a, 0x2c, 0xe1, 0xe2, 0x3d, 0xb9, 0x4a, 0xef, 0x39, 0xde, 0x29,
	0x76, 0xb0, 0xf8, 0x4b, 0xa2, 0xb6, 0x80, 0x87, 0x6c, 0x48, 0xd7, 0x23, 0xf5, 0x7d, 0x84, 0x1b,
	0x40, 0xba, 0xa5, 0x5d, 0xef, 0x94, 0x6e, 0x3d, 0xb3, 0x55, 0x63, 0xf3, 0x49, 0x91, 0xae, 0xa7,
	0xf5, 0x7f, 0x76, 0xc4, 0x17, 0x58, 0x7c, 0xa2, 0x9b, 0x2d, 0x75, 0xf6, 0xa0, 0xcc, 0x75, 0xc7,
	0xbc, 0xc7, 0x4b, 0x98, 0x2a, 0xdd, 0xd0, 0x6d, 0xd4, 0x87, 0x02, 0x11, 0xce, 0x3a, 0x66, 0x17,
	0x5f, 0xed, 0xd7, 0x43, 0xa7, 0x19, 0x8e, 0xe4, 0x69, 0x91, 0xae, 0xb3, 0x3a, 0x14, 0x42, 0xc2,
	0xfc, 0x9a, 0xc9, 0x05, 0xd9, 0x12, 0x52, 0x73, 0x50, 0x31, 0xb2, 0x61, 0x89, 0x02, 0x32, 0xd3,
	0xf1, 0x89, 0xf4, 0x47, 0x92, 0x27, 0x0a, 0x6f, 0xcb, 0xea, 0x11, 0xc3, 0x67, 0x00, 0xde, 0xf5,
	0x99, 0x1b, 0xb2, 0xd1, 0x7e, 0x8f, 0xbc, 0xfa, 0x99, 0xc0, 0xd9, 0x70, 0x07, 0x7e, 0x80, 0x59,
	0x48, 0x1e, 0x71, 0x33, 0x4c, 0x78, 0x34, 0xb5, 0xd5, 0xe3, 0x11, 0x0b, 0x19, 0x8a, 0x27, 0xdf,
	0x7f, 0xfd, 0xfe, 0x31, 0x59, 0x08, 0x28, 0x4f, 0x2f, 0x4b, 0xeb, 0xf7, 0xde, 0x24, 0xcf, 0xf1,
	0x1d, 0x9c, 0xc7, 0xb4, 0x31, 0x1c, 0x1b, 0x4f, 0x63, 0x75, 0x39, 0x86, 0x51, 0xb6, 0xf4, 0x32,
	0xc0, 0x07, 0x83, 0x4c, 0xe9, 0x3d, 0x6f, 0x67, 0xfe, 0x07, 0xbd, 0xfe, 0x13, 0x00, 0x00, 0xff,
	0xff, 0x0b, 0x55, 0xa0, 0x1c, 0x71, 0x02, 0x00, 0x00,
}
