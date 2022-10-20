// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        (unknown)
// source: rpccore/apicore/apicore.proto

package apicore

import (
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ComputeRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	D *DagParams `protobuf:"bytes,1,opt,name=d,proto3" json:"d,omitempty"`
}

func (x *ComputeRequest) Reset() {
	*x = ComputeRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpccore_apicore_apicore_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ComputeRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ComputeRequest) ProtoMessage() {}

func (x *ComputeRequest) ProtoReflect() protoreflect.Message {
	mi := &file_rpccore_apicore_apicore_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ComputeRequest.ProtoReflect.Descriptor instead.
func (*ComputeRequest) Descriptor() ([]byte, []int) {
	return file_rpccore_apicore_apicore_proto_rawDescGZIP(), []int{0}
}

func (x *ComputeRequest) GetD() *DagParams {
	if x != nil {
		return x.D
	}
	return nil
}

type ComputeResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Phi []byte `protobuf:"bytes,1,opt,name=phi,proto3" json:"phi,omitempty"`
}

func (x *ComputeResponse) Reset() {
	*x = ComputeResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpccore_apicore_apicore_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ComputeResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ComputeResponse) ProtoMessage() {}

func (x *ComputeResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpccore_apicore_apicore_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ComputeResponse.ProtoReflect.Descriptor instead.
func (*ComputeResponse) Descriptor() ([]byte, []int) {
	return file_rpccore_apicore_apicore_proto_rawDescGZIP(), []int{1}
}

func (x *ComputeResponse) GetPhi() []byte {
	if x != nil {
		return x.Phi
	}
	return nil
}

type GetNIPRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetNIPRequest) Reset() {
	*x = GetNIPRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpccore_apicore_apicore_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetNIPRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetNIPRequest) ProtoMessage() {}

func (x *GetNIPRequest) ProtoReflect() protoreflect.Message {
	mi := &file_rpccore_apicore_apicore_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetNIPRequest.ProtoReflect.Descriptor instead.
func (*GetNIPRequest) Descriptor() ([]byte, []int) {
	return file_rpccore_apicore_apicore_proto_rawDescGZIP(), []int{2}
}

type GetNIPResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Proof *Proof `protobuf:"bytes,1,opt,name=proof,proto3" json:"proof,omitempty"`
}

func (x *GetNIPResponse) Reset() {
	*x = GetNIPResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpccore_apicore_apicore_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetNIPResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetNIPResponse) ProtoMessage() {}

func (x *GetNIPResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpccore_apicore_apicore_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetNIPResponse.ProtoReflect.Descriptor instead.
func (*GetNIPResponse) Descriptor() ([]byte, []int) {
	return file_rpccore_apicore_apicore_proto_rawDescGZIP(), []int{3}
}

func (x *GetNIPResponse) GetProof() *Proof {
	if x != nil {
		return x.Proof
	}
	return nil
}

type ShutdownRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ShutdownRequest) Reset() {
	*x = ShutdownRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpccore_apicore_apicore_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ShutdownRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShutdownRequest) ProtoMessage() {}

func (x *ShutdownRequest) ProtoReflect() protoreflect.Message {
	mi := &file_rpccore_apicore_apicore_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ShutdownRequest.ProtoReflect.Descriptor instead.
func (*ShutdownRequest) Descriptor() ([]byte, []int) {
	return file_rpccore_apicore_apicore_proto_rawDescGZIP(), []int{4}
}

type ShutdownResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ShutdownResponse) Reset() {
	*x = ShutdownResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpccore_apicore_apicore_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ShutdownResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShutdownResponse) ProtoMessage() {}

func (x *ShutdownResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpccore_apicore_apicore_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ShutdownResponse.ProtoReflect.Descriptor instead.
func (*ShutdownResponse) Descriptor() ([]byte, []int) {
	return file_rpccore_apicore_apicore_proto_rawDescGZIP(), []int{5}
}

type VerifyNIPRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	D *DagParams `protobuf:"bytes,1,opt,name=d,proto3" json:"d,omitempty"`
	P *Proof     `protobuf:"bytes,2,opt,name=p,proto3" json:"p,omitempty"`
}

func (x *VerifyNIPRequest) Reset() {
	*x = VerifyNIPRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpccore_apicore_apicore_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VerifyNIPRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VerifyNIPRequest) ProtoMessage() {}

func (x *VerifyNIPRequest) ProtoReflect() protoreflect.Message {
	mi := &file_rpccore_apicore_apicore_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VerifyNIPRequest.ProtoReflect.Descriptor instead.
func (*VerifyNIPRequest) Descriptor() ([]byte, []int) {
	return file_rpccore_apicore_apicore_proto_rawDescGZIP(), []int{6}
}

func (x *VerifyNIPRequest) GetD() *DagParams {
	if x != nil {
		return x.D
	}
	return nil
}

func (x *VerifyNIPRequest) GetP() *Proof {
	if x != nil {
		return x.P
	}
	return nil
}

type VerifyNIPResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Verified bool `protobuf:"varint,1,opt,name=verified,proto3" json:"verified,omitempty"`
}

func (x *VerifyNIPResponse) Reset() {
	*x = VerifyNIPResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpccore_apicore_apicore_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VerifyNIPResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VerifyNIPResponse) ProtoMessage() {}

func (x *VerifyNIPResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpccore_apicore_apicore_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VerifyNIPResponse.ProtoReflect.Descriptor instead.
func (*VerifyNIPResponse) Descriptor() ([]byte, []int) {
	return file_rpccore_apicore_apicore_proto_rawDescGZIP(), []int{7}
}

func (x *VerifyNIPResponse) GetVerified() bool {
	if x != nil {
		return x.Verified
	}
	return false
}

type DagParams struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	X []byte `protobuf:"bytes,1,opt,name=x,proto3" json:"x,omitempty"`
	N uint32 `protobuf:"varint,2,opt,name=n,proto3" json:"n,omitempty"`
}

func (x *DagParams) Reset() {
	*x = DagParams{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpccore_apicore_apicore_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DagParams) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DagParams) ProtoMessage() {}

func (x *DagParams) ProtoReflect() protoreflect.Message {
	mi := &file_rpccore_apicore_apicore_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DagParams.ProtoReflect.Descriptor instead.
func (*DagParams) Descriptor() ([]byte, []int) {
	return file_rpccore_apicore_apicore_proto_rawDescGZIP(), []int{8}
}

func (x *DagParams) GetX() []byte {
	if x != nil {
		return x.X
	}
	return nil
}

func (x *DagParams) GetN() uint32 {
	if x != nil {
		return x.N
	}
	return 0
}

type Proof struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Phi          []byte   `protobuf:"bytes,1,opt,name=phi,proto3" json:"phi,omitempty"`
	ProvenLeaves [][]byte `protobuf:"bytes,2,rep,name=provenLeaves,json=proven_leaves,proto3" json:"provenLeaves,omitempty"`
	ProofNodes   [][]byte `protobuf:"bytes,3,rep,name=proofNodes,json=proof_nodes,proto3" json:"proofNodes,omitempty"`
}

func (x *Proof) Reset() {
	*x = Proof{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpccore_apicore_apicore_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Proof) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Proof) ProtoMessage() {}

func (x *Proof) ProtoReflect() protoreflect.Message {
	mi := &file_rpccore_apicore_apicore_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Proof.ProtoReflect.Descriptor instead.
func (*Proof) Descriptor() ([]byte, []int) {
	return file_rpccore_apicore_apicore_proto_rawDescGZIP(), []int{9}
}

func (x *Proof) GetPhi() []byte {
	if x != nil {
		return x.Phi
	}
	return nil
}

func (x *Proof) GetProvenLeaves() [][]byte {
	if x != nil {
		return x.ProvenLeaves
	}
	return nil
}

func (x *Proof) GetProofNodes() [][]byte {
	if x != nil {
		return x.ProofNodes
	}
	return nil
}

var File_rpccore_apicore_apicore_proto protoreflect.FileDescriptor

var file_rpccore_apicore_apicore_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x72, 0x70, 0x63, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72,
	0x65, 0x2f, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x07, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x32, 0x0a, 0x0e, 0x43, 0x6f, 0x6d, 0x70, 0x75, 0x74,
	0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x20, 0x0a, 0x01, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x44, 0x61,
	0x67, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x52, 0x01, 0x64, 0x22, 0x23, 0x0a, 0x0f, 0x43, 0x6f,
	0x6d, 0x70, 0x75, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x10, 0x0a,
	0x03, 0x70, 0x68, 0x69, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x03, 0x70, 0x68, 0x69, 0x22,
	0x0f, 0x0a, 0x0d, 0x47, 0x65, 0x74, 0x4e, 0x49, 0x50, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x22, 0x36, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x4e, 0x49, 0x50, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x24, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x0e, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x50, 0x72, 0x6f, 0x6f,
	0x66, 0x52, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x22, 0x11, 0x0a, 0x0f, 0x53, 0x68, 0x75, 0x74,
	0x64, 0x6f, 0x77, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x12, 0x0a, 0x10, 0x53,
	0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x52, 0x0a, 0x10, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x4e, 0x49, 0x50, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x20, 0x0a, 0x01, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12,
	0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x44, 0x61, 0x67, 0x50, 0x61, 0x72, 0x61,
	0x6d, 0x73, 0x52, 0x01, 0x64, 0x12, 0x1c, 0x0a, 0x01, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x0e, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x50, 0x72, 0x6f, 0x6f, 0x66,
	0x52, 0x01, 0x70, 0x22, 0x2f, 0x0a, 0x11, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x4e, 0x49, 0x50,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x76, 0x65, 0x72, 0x69,
	0x66, 0x69, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x76, 0x65, 0x72, 0x69,
	0x66, 0x69, 0x65, 0x64, 0x22, 0x27, 0x0a, 0x09, 0x44, 0x61, 0x67, 0x50, 0x61, 0x72, 0x61, 0x6d,
	0x73, 0x12, 0x0c, 0x0a, 0x01, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x01, 0x78, 0x12,
	0x0c, 0x0a, 0x01, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x01, 0x6e, 0x22, 0x5f, 0x0a,
	0x05, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x10, 0x0a, 0x03, 0x70, 0x68, 0x69, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x03, 0x70, 0x68, 0x69, 0x12, 0x23, 0x0a, 0x0c, 0x70, 0x72, 0x6f, 0x76,
	0x65, 0x6e, 0x4c, 0x65, 0x61, 0x76, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x0d,
	0x70, 0x72, 0x6f, 0x76, 0x65, 0x6e, 0x5f, 0x6c, 0x65, 0x61, 0x76, 0x65, 0x73, 0x12, 0x1f, 0x0a,
	0x0a, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x4e, 0x6f, 0x64, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28,
	0x0c, 0x52, 0x0b, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x5f, 0x6e, 0x6f, 0x64, 0x65, 0x73, 0x32, 0x9e,
	0x02, 0x0a, 0x0e, 0x50, 0x6f, 0x65, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x50, 0x72, 0x6f, 0x76, 0x65,
	0x72, 0x12, 0x58, 0x0a, 0x07, 0x43, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x65, 0x12, 0x17, 0x2e, 0x61,
	0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x18, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e,
	0x43, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x1a, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x14, 0x12, 0x12, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x72, 0x6f,
	0x76, 0x65, 0x72, 0x2f, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x65, 0x12, 0x54, 0x0a, 0x06, 0x47,
	0x65, 0x74, 0x4e, 0x49, 0x50, 0x12, 0x16, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e,
	0x47, 0x65, 0x74, 0x4e, 0x49, 0x50, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e,
	0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x4e, 0x49, 0x50, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x19, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x13, 0x12, 0x11,
	0x2f, 0x76, 0x31, 0x2f, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x2f, 0x67, 0x65, 0x74, 0x6e, 0x69,
	0x70, 0x12, 0x5c, 0x0a, 0x08, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x12, 0x18, 0x2e,
	0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72,
	0x65, 0x2e, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x1b, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x15, 0x12, 0x13, 0x2f, 0x76, 0x31, 0x2f,
	0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x2f, 0x73, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x32,
	0x72, 0x0a, 0x0c, 0x50, 0x6f, 0x65, 0x74, 0x56, 0x65, 0x72, 0x69, 0x66, 0x69, 0x65, 0x72, 0x12,
	0x62, 0x0a, 0x09, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x4e, 0x49, 0x50, 0x12, 0x19, 0x2e, 0x61,
	0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x4e, 0x49, 0x50,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1a, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72,
	0x65, 0x2e, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x4e, 0x49, 0x50, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x1e, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x18, 0x12, 0x16, 0x2f, 0x76, 0x31,
	0x2f, 0x76, 0x65, 0x72, 0x69, 0x66, 0x69, 0x65, 0x72, 0x2f, 0x76, 0x65, 0x72, 0x69, 0x66, 0x79,
	0x6e, 0x69, 0x70, 0x42, 0x91, 0x01, 0x0a, 0x0b, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x70, 0x69, 0x63,
	0x6f, 0x72, 0x65, 0x42, 0x0c, 0x41, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x50, 0x72, 0x6f, 0x74,
	0x6f, 0x50, 0x01, 0x5a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x73, 0x70, 0x61, 0x63, 0x65, 0x6d, 0x65, 0x73, 0x68, 0x6f, 0x73, 0x2f, 0x70, 0x6f, 0x65, 0x74,
	0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x72, 0x70,
	0x63, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0xa2, 0x02, 0x03,
	0x41, 0x58, 0x58, 0xaa, 0x02, 0x07, 0x41, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0xca, 0x02, 0x07,
	0x41, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0xe2, 0x02, 0x13, 0x41, 0x70, 0x69, 0x63, 0x6f, 0x72,
	0x65, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x07,
	0x41, 0x70, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_rpccore_apicore_apicore_proto_rawDescOnce sync.Once
	file_rpccore_apicore_apicore_proto_rawDescData = file_rpccore_apicore_apicore_proto_rawDesc
)

func file_rpccore_apicore_apicore_proto_rawDescGZIP() []byte {
	file_rpccore_apicore_apicore_proto_rawDescOnce.Do(func() {
		file_rpccore_apicore_apicore_proto_rawDescData = protoimpl.X.CompressGZIP(file_rpccore_apicore_apicore_proto_rawDescData)
	})
	return file_rpccore_apicore_apicore_proto_rawDescData
}

var file_rpccore_apicore_apicore_proto_msgTypes = make([]protoimpl.MessageInfo, 10)
var file_rpccore_apicore_apicore_proto_goTypes = []interface{}{
	(*ComputeRequest)(nil),    // 0: apicore.ComputeRequest
	(*ComputeResponse)(nil),   // 1: apicore.ComputeResponse
	(*GetNIPRequest)(nil),     // 2: apicore.GetNIPRequest
	(*GetNIPResponse)(nil),    // 3: apicore.GetNIPResponse
	(*ShutdownRequest)(nil),   // 4: apicore.ShutdownRequest
	(*ShutdownResponse)(nil),  // 5: apicore.ShutdownResponse
	(*VerifyNIPRequest)(nil),  // 6: apicore.VerifyNIPRequest
	(*VerifyNIPResponse)(nil), // 7: apicore.VerifyNIPResponse
	(*DagParams)(nil),         // 8: apicore.DagParams
	(*Proof)(nil),             // 9: apicore.Proof
}
var file_rpccore_apicore_apicore_proto_depIdxs = []int32{
	8, // 0: apicore.ComputeRequest.d:type_name -> apicore.DagParams
	9, // 1: apicore.GetNIPResponse.proof:type_name -> apicore.Proof
	8, // 2: apicore.VerifyNIPRequest.d:type_name -> apicore.DagParams
	9, // 3: apicore.VerifyNIPRequest.p:type_name -> apicore.Proof
	0, // 4: apicore.PoetCoreProver.Compute:input_type -> apicore.ComputeRequest
	2, // 5: apicore.PoetCoreProver.GetNIP:input_type -> apicore.GetNIPRequest
	4, // 6: apicore.PoetCoreProver.Shutdown:input_type -> apicore.ShutdownRequest
	6, // 7: apicore.PoetVerifier.VerifyNIP:input_type -> apicore.VerifyNIPRequest
	1, // 8: apicore.PoetCoreProver.Compute:output_type -> apicore.ComputeResponse
	3, // 9: apicore.PoetCoreProver.GetNIP:output_type -> apicore.GetNIPResponse
	5, // 10: apicore.PoetCoreProver.Shutdown:output_type -> apicore.ShutdownResponse
	7, // 11: apicore.PoetVerifier.VerifyNIP:output_type -> apicore.VerifyNIPResponse
	8, // [8:12] is the sub-list for method output_type
	4, // [4:8] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_rpccore_apicore_apicore_proto_init() }
func file_rpccore_apicore_apicore_proto_init() {
	if File_rpccore_apicore_apicore_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_rpccore_apicore_apicore_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ComputeRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpccore_apicore_apicore_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ComputeResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpccore_apicore_apicore_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetNIPRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpccore_apicore_apicore_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetNIPResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpccore_apicore_apicore_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ShutdownRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpccore_apicore_apicore_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ShutdownResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpccore_apicore_apicore_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VerifyNIPRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpccore_apicore_apicore_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VerifyNIPResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpccore_apicore_apicore_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DagParams); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpccore_apicore_apicore_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Proof); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_rpccore_apicore_apicore_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   10,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_rpccore_apicore_apicore_proto_goTypes,
		DependencyIndexes: file_rpccore_apicore_apicore_proto_depIdxs,
		MessageInfos:      file_rpccore_apicore_apicore_proto_msgTypes,
	}.Build()
	File_rpccore_apicore_apicore_proto = out.File
	file_rpccore_apicore_apicore_proto_rawDesc = nil
	file_rpccore_apicore_apicore_proto_goTypes = nil
	file_rpccore_apicore_apicore_proto_depIdxs = nil
}
