// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        (unknown)
// source: rpc/api/api.proto

package api

import (
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type StartRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GatewayAddresses       []string `protobuf:"bytes,1,rep,name=gatewayAddresses,proto3" json:"gatewayAddresses,omitempty"`
	DisableBroadcast       bool     `protobuf:"varint,2,opt,name=disableBroadcast,proto3" json:"disableBroadcast,omitempty"`
	ConnAcksThreshold      int32    `protobuf:"varint,3,opt,name=connAcksThreshold,proto3" json:"connAcksThreshold,omitempty"`
	BroadcastAcksThreshold int32    `protobuf:"varint,4,opt,name=broadcastAcksThreshold,proto3" json:"broadcastAcksThreshold,omitempty"`
}

func (x *StartRequest) Reset() {
	*x = StartRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartRequest) ProtoMessage() {}

func (x *StartRequest) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartRequest.ProtoReflect.Descriptor instead.
func (*StartRequest) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{0}
}

func (x *StartRequest) GetGatewayAddresses() []string {
	if x != nil {
		return x.GatewayAddresses
	}
	return nil
}

func (x *StartRequest) GetDisableBroadcast() bool {
	if x != nil {
		return x.DisableBroadcast
	}
	return false
}

func (x *StartRequest) GetConnAcksThreshold() int32 {
	if x != nil {
		return x.ConnAcksThreshold
	}
	return 0
}

func (x *StartRequest) GetBroadcastAcksThreshold() int32 {
	if x != nil {
		return x.BroadcastAcksThreshold
	}
	return 0
}

type StartResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *StartResponse) Reset() {
	*x = StartResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartResponse) ProtoMessage() {}

func (x *StartResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartResponse.ProtoReflect.Descriptor instead.
func (*StartResponse) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{1}
}

type UpdateGatewayRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GatewayAddresses       []string `protobuf:"bytes,1,rep,name=gatewayAddresses,proto3" json:"gatewayAddresses,omitempty"`
	DisableBroadcast       bool     `protobuf:"varint,2,opt,name=disableBroadcast,proto3" json:"disableBroadcast,omitempty"`
	ConnAcksThreshold      int32    `protobuf:"varint,3,opt,name=connAcksThreshold,proto3" json:"connAcksThreshold,omitempty"`
	BroadcastAcksThreshold int32    `protobuf:"varint,4,opt,name=broadcastAcksThreshold,proto3" json:"broadcastAcksThreshold,omitempty"`
}

func (x *UpdateGatewayRequest) Reset() {
	*x = UpdateGatewayRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateGatewayRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateGatewayRequest) ProtoMessage() {}

func (x *UpdateGatewayRequest) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateGatewayRequest.ProtoReflect.Descriptor instead.
func (*UpdateGatewayRequest) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{2}
}

func (x *UpdateGatewayRequest) GetGatewayAddresses() []string {
	if x != nil {
		return x.GatewayAddresses
	}
	return nil
}

func (x *UpdateGatewayRequest) GetDisableBroadcast() bool {
	if x != nil {
		return x.DisableBroadcast
	}
	return false
}

func (x *UpdateGatewayRequest) GetConnAcksThreshold() int32 {
	if x != nil {
		return x.ConnAcksThreshold
	}
	return 0
}

func (x *UpdateGatewayRequest) GetBroadcastAcksThreshold() int32 {
	if x != nil {
		return x.BroadcastAcksThreshold
	}
	return 0
}

type UpdateGatewayResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *UpdateGatewayResponse) Reset() {
	*x = UpdateGatewayResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateGatewayResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateGatewayResponse) ProtoMessage() {}

func (x *UpdateGatewayResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateGatewayResponse.ProtoReflect.Descriptor instead.
func (*UpdateGatewayResponse) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{3}
}

type SubmitRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Challenge []byte `protobuf:"bytes,1,opt,name=challenge,proto3" json:"challenge,omitempty"`
	Signature []byte `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (x *SubmitRequest) Reset() {
	*x = SubmitRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SubmitRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SubmitRequest) ProtoMessage() {}

func (x *SubmitRequest) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SubmitRequest.ProtoReflect.Descriptor instead.
func (*SubmitRequest) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{4}
}

func (x *SubmitRequest) GetChallenge() []byte {
	if x != nil {
		return x.Challenge
	}
	return nil
}

func (x *SubmitRequest) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

type SubmitResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RoundId  string               `protobuf:"bytes,1,opt,name=roundId,proto3" json:"roundId,omitempty"`
	Hash     []byte               `protobuf:"bytes,2,opt,name=hash,proto3" json:"hash,omitempty"`
	RoundEnd *durationpb.Duration `protobuf:"bytes,3,opt,name=round_end,json=roundEnd,proto3" json:"round_end,omitempty"`
}

func (x *SubmitResponse) Reset() {
	*x = SubmitResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SubmitResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SubmitResponse) ProtoMessage() {}

func (x *SubmitResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SubmitResponse.ProtoReflect.Descriptor instead.
func (*SubmitResponse) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{5}
}

func (x *SubmitResponse) GetRoundId() string {
	if x != nil {
		return x.RoundId
	}
	return ""
}

func (x *SubmitResponse) GetHash() []byte {
	if x != nil {
		return x.Hash
	}
	return nil
}

func (x *SubmitResponse) GetRoundEnd() *durationpb.Duration {
	if x != nil {
		return x.RoundEnd
	}
	return nil
}

type GetInfoRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetInfoRequest) Reset() {
	*x = GetInfoRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetInfoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetInfoRequest) ProtoMessage() {}

func (x *GetInfoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetInfoRequest.ProtoReflect.Descriptor instead.
func (*GetInfoRequest) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{6}
}

type GetInfoResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	OpenRoundId        string   `protobuf:"bytes,1,opt,name=openRoundId,proto3" json:"openRoundId,omitempty"`
	ExecutingRoundsIds []string `protobuf:"bytes,2,rep,name=executingRoundsIds,proto3" json:"executingRoundsIds,omitempty"`
	ServicePubKey      []byte   `protobuf:"bytes,3,opt,name=servicePubKey,proto3" json:"servicePubKey,omitempty"`
}

func (x *GetInfoResponse) Reset() {
	*x = GetInfoResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetInfoResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetInfoResponse) ProtoMessage() {}

func (x *GetInfoResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetInfoResponse.ProtoReflect.Descriptor instead.
func (*GetInfoResponse) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{7}
}

func (x *GetInfoResponse) GetOpenRoundId() string {
	if x != nil {
		return x.OpenRoundId
	}
	return ""
}

func (x *GetInfoResponse) GetExecutingRoundsIds() []string {
	if x != nil {
		return x.ExecutingRoundsIds
	}
	return nil
}

func (x *GetInfoResponse) GetServicePubKey() []byte {
	if x != nil {
		return x.ServicePubKey
	}
	return nil
}

type MembershipProof struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Index int32    `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"`
	Root  []byte   `protobuf:"bytes,2,opt,name=root,proto3" json:"root,omitempty"`
	Proof [][]byte `protobuf:"bytes,3,rep,name=proof,proto3" json:"proof,omitempty"`
}

func (x *MembershipProof) Reset() {
	*x = MembershipProof{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MembershipProof) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MembershipProof) ProtoMessage() {}

func (x *MembershipProof) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MembershipProof.ProtoReflect.Descriptor instead.
func (*MembershipProof) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{8}
}

func (x *MembershipProof) GetIndex() int32 {
	if x != nil {
		return x.Index
	}
	return 0
}

func (x *MembershipProof) GetRoot() []byte {
	if x != nil {
		return x.Root
	}
	return nil
}

func (x *MembershipProof) GetProof() [][]byte {
	if x != nil {
		return x.Proof
	}
	return nil
}

type MerkleProof struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Root         []byte   `protobuf:"bytes,1,opt,name=root,proto3" json:"root,omitempty"`
	ProvenLeaves [][]byte `protobuf:"bytes,2,rep,name=proven_leaves,json=provenLeaves,proto3" json:"proven_leaves,omitempty"`
	ProofNodes   [][]byte `protobuf:"bytes,3,rep,name=proof_nodes,json=proofNodes,proto3" json:"proof_nodes,omitempty"`
}

func (x *MerkleProof) Reset() {
	*x = MerkleProof{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MerkleProof) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MerkleProof) ProtoMessage() {}

func (x *MerkleProof) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MerkleProof.ProtoReflect.Descriptor instead.
func (*MerkleProof) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{9}
}

func (x *MerkleProof) GetRoot() []byte {
	if x != nil {
		return x.Root
	}
	return nil
}

func (x *MerkleProof) GetProvenLeaves() [][]byte {
	if x != nil {
		return x.ProvenLeaves
	}
	return nil
}

func (x *MerkleProof) GetProofNodes() [][]byte {
	if x != nil {
		return x.ProofNodes
	}
	return nil
}

type PoetProof struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Proof   *MerkleProof `protobuf:"bytes,1,opt,name=proof,proto3" json:"proof,omitempty"`
	Members [][]byte     `protobuf:"bytes,2,rep,name=members,proto3" json:"members,omitempty"`
	Leaves  uint64       `protobuf:"varint,3,opt,name=leaves,proto3" json:"leaves,omitempty"`
}

func (x *PoetProof) Reset() {
	*x = PoetProof{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PoetProof) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PoetProof) ProtoMessage() {}

func (x *PoetProof) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PoetProof.ProtoReflect.Descriptor instead.
func (*PoetProof) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{10}
}

func (x *PoetProof) GetProof() *MerkleProof {
	if x != nil {
		return x.Proof
	}
	return nil
}

func (x *PoetProof) GetMembers() [][]byte {
	if x != nil {
		return x.Members
	}
	return nil
}

func (x *PoetProof) GetLeaves() uint64 {
	if x != nil {
		return x.Leaves
	}
	return 0
}

type GetProofRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RoundId string `protobuf:"bytes,1,opt,name=round_id,json=roundId,proto3" json:"round_id,omitempty"`
}

func (x *GetProofRequest) Reset() {
	*x = GetProofRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[11]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetProofRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetProofRequest) ProtoMessage() {}

func (x *GetProofRequest) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[11]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetProofRequest.ProtoReflect.Descriptor instead.
func (*GetProofRequest) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{11}
}

func (x *GetProofRequest) GetRoundId() string {
	if x != nil {
		return x.RoundId
	}
	return ""
}

type GetProofResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Proof  *PoetProof `protobuf:"bytes,1,opt,name=proof,proto3" json:"proof,omitempty"`
	Pubkey []byte     `protobuf:"bytes,2,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
}

func (x *GetProofResponse) Reset() {
	*x = GetProofResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_api_api_proto_msgTypes[12]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetProofResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetProofResponse) ProtoMessage() {}

func (x *GetProofResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_api_api_proto_msgTypes[12]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetProofResponse.ProtoReflect.Descriptor instead.
func (*GetProofResponse) Descriptor() ([]byte, []int) {
	return file_rpc_api_api_proto_rawDescGZIP(), []int{12}
}

func (x *GetProofResponse) GetProof() *PoetProof {
	if x != nil {
		return x.Proof
	}
	return nil
}

func (x *GetProofResponse) GetPubkey() []byte {
	if x != nil {
		return x.Pubkey
	}
	return nil
}

var File_rpc_api_api_proto protoreflect.FileDescriptor

var file_rpc_api_api_proto_rawDesc = []byte{
	0x0a, 0x11, 0x72, 0x70, 0x63, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x03, 0x61, 0x70, 0x69, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xcc, 0x01, 0x0a, 0x0c, 0x53, 0x74, 0x61, 0x72, 0x74,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2a, 0x0a, 0x10, 0x67, 0x61, 0x74, 0x65, 0x77,
	0x61, 0x79, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x10, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x65, 0x73, 0x12, 0x2a, 0x0a, 0x10, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x42, 0x72,
	0x6f, 0x61, 0x64, 0x63, 0x61, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x10, 0x64,
	0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x42, 0x72, 0x6f, 0x61, 0x64, 0x63, 0x61, 0x73, 0x74, 0x12,
	0x2c, 0x0a, 0x11, 0x63, 0x6f, 0x6e, 0x6e, 0x41, 0x63, 0x6b, 0x73, 0x54, 0x68, 0x72, 0x65, 0x73,
	0x68, 0x6f, 0x6c, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x11, 0x63, 0x6f, 0x6e, 0x6e,
	0x41, 0x63, 0x6b, 0x73, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x12, 0x36, 0x0a,
	0x16, 0x62, 0x72, 0x6f, 0x61, 0x64, 0x63, 0x61, 0x73, 0x74, 0x41, 0x63, 0x6b, 0x73, 0x54, 0x68,
	0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x16, 0x62,
	0x72, 0x6f, 0x61, 0x64, 0x63, 0x61, 0x73, 0x74, 0x41, 0x63, 0x6b, 0x73, 0x54, 0x68, 0x72, 0x65,
	0x73, 0x68, 0x6f, 0x6c, 0x64, 0x22, 0x0f, 0x0a, 0x0d, 0x53, 0x74, 0x61, 0x72, 0x74, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0xd4, 0x01, 0x0a, 0x14, 0x55, 0x70, 0x64, 0x61, 0x74,
	0x65, 0x47, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x2a, 0x0a, 0x10, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x10, 0x67, 0x61, 0x74, 0x65, 0x77,
	0x61, 0x79, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x65, 0x73, 0x12, 0x2a, 0x0a, 0x10, 0x64,
	0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x42, 0x72, 0x6f, 0x61, 0x64, 0x63, 0x61, 0x73, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x10, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x42, 0x72,
	0x6f, 0x61, 0x64, 0x63, 0x61, 0x73, 0x74, 0x12, 0x2c, 0x0a, 0x11, 0x63, 0x6f, 0x6e, 0x6e, 0x41,
	0x63, 0x6b, 0x73, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x11, 0x63, 0x6f, 0x6e, 0x6e, 0x41, 0x63, 0x6b, 0x73, 0x54, 0x68, 0x72, 0x65,
	0x73, 0x68, 0x6f, 0x6c, 0x64, 0x12, 0x36, 0x0a, 0x16, 0x62, 0x72, 0x6f, 0x61, 0x64, 0x63, 0x61,
	0x73, 0x74, 0x41, 0x63, 0x6b, 0x73, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x16, 0x62, 0x72, 0x6f, 0x61, 0x64, 0x63, 0x61, 0x73, 0x74,
	0x41, 0x63, 0x6b, 0x73, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x22, 0x17, 0x0a,
	0x15, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x47, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x4b, 0x0a, 0x0d, 0x53, 0x75, 0x62, 0x6d, 0x69, 0x74,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x63, 0x68, 0x61, 0x6c, 0x6c,
	0x65, 0x6e, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x63, 0x68, 0x61, 0x6c,
	0x6c, 0x65, 0x6e, 0x67, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75,
	0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74,
	0x75, 0x72, 0x65, 0x22, 0x76, 0x0a, 0x0e, 0x53, 0x75, 0x62, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x49, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x49, 0x64, 0x12,
	0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x68,
	0x61, 0x73, 0x68, 0x12, 0x36, 0x0a, 0x09, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x5f, 0x65, 0x6e, 0x64,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x52, 0x08, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x45, 0x6e, 0x64, 0x22, 0x10, 0x0a, 0x0e, 0x47,
	0x65, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x89, 0x01,
	0x0a, 0x0f, 0x47, 0x65, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x20, 0x0a, 0x0b, 0x6f, 0x70, 0x65, 0x6e, 0x52, 0x6f, 0x75, 0x6e, 0x64, 0x49, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x6f, 0x70, 0x65, 0x6e, 0x52, 0x6f, 0x75, 0x6e,
	0x64, 0x49, 0x64, 0x12, 0x2e, 0x0a, 0x12, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6e, 0x67,
	0x52, 0x6f, 0x75, 0x6e, 0x64, 0x73, 0x49, 0x64, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52,
	0x12, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6e, 0x67, 0x52, 0x6f, 0x75, 0x6e, 0x64, 0x73,
	0x49, 0x64, 0x73, 0x12, 0x24, 0x0a, 0x0d, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x75,
	0x62, 0x4b, 0x65, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0d, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x50, 0x75, 0x62, 0x4b, 0x65, 0x79, 0x22, 0x51, 0x0a, 0x0f, 0x4d, 0x65, 0x6d,
	0x62, 0x65, 0x72, 0x73, 0x68, 0x69, 0x70, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x14, 0x0a, 0x05,
	0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x69, 0x6e, 0x64,
	0x65, 0x78, 0x12, 0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18,
	0x03, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x22, 0x67, 0x0a, 0x0b,
	0x4d, 0x65, 0x72, 0x6b, 0x6c, 0x65, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x12, 0x0a, 0x04, 0x72,
	0x6f, 0x6f, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x12,
	0x23, 0x0a, 0x0d, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x6e, 0x5f, 0x6c, 0x65, 0x61, 0x76, 0x65, 0x73,
	0x18, 0x02, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x0c, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x6e, 0x4c, 0x65,
	0x61, 0x76, 0x65, 0x73, 0x12, 0x1f, 0x0a, 0x0b, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x5f, 0x6e, 0x6f,
	0x64, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x0a, 0x70, 0x72, 0x6f, 0x6f, 0x66,
	0x4e, 0x6f, 0x64, 0x65, 0x73, 0x22, 0x65, 0x0a, 0x09, 0x50, 0x6f, 0x65, 0x74, 0x50, 0x72, 0x6f,
	0x6f, 0x66, 0x12, 0x26, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x10, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x4d, 0x65, 0x72, 0x6b, 0x6c, 0x65, 0x50, 0x72,
	0x6f, 0x6f, 0x66, 0x52, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65,
	0x6d, 0x62, 0x65, 0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x07, 0x6d, 0x65, 0x6d,
	0x62, 0x65, 0x72, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x6c, 0x65, 0x61, 0x76, 0x65, 0x73, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x6c, 0x65, 0x61, 0x76, 0x65, 0x73, 0x22, 0x2c, 0x0a, 0x0f,
	0x47, 0x65, 0x74, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x19, 0x0a, 0x08, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x49, 0x64, 0x22, 0x50, 0x0a, 0x10, 0x47, 0x65,
	0x74, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x24,
	0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x50, 0x6f, 0x65, 0x74, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x05, 0x70,
	0x72, 0x6f, 0x6f, 0x66, 0x12, 0x16, 0x0a, 0x06, 0x70, 0x75, 0x62, 0x6b, 0x65, 0x79, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x06, 0x70, 0x75, 0x62, 0x6b, 0x65, 0x79, 0x32, 0x9c, 0x03, 0x0a,
	0x04, 0x50, 0x6f, 0x65, 0x74, 0x12, 0x44, 0x0a, 0x05, 0x53, 0x74, 0x61, 0x72, 0x74, 0x12, 0x11,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x53, 0x74, 0x61, 0x72, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x53, 0x74, 0x61, 0x72, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x14, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x0e, 0x22, 0x09, 0x2f,
	0x76, 0x31, 0x2f, 0x73, 0x74, 0x61, 0x72, 0x74, 0x3a, 0x01, 0x2a, 0x12, 0x64, 0x0a, 0x0d, 0x55,
	0x70, 0x64, 0x61, 0x74, 0x65, 0x47, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x12, 0x19, 0x2e, 0x61,
	0x70, 0x69, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x47, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1a, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x55, 0x70,
	0x64, 0x61, 0x74, 0x65, 0x47, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x1c, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x16, 0x22, 0x11, 0x2f, 0x76, 0x31,
	0x2f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x3a, 0x01,
	0x2a, 0x12, 0x48, 0x0a, 0x06, 0x53, 0x75, 0x62, 0x6d, 0x69, 0x74, 0x12, 0x12, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x53, 0x75, 0x62, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x13, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x53, 0x75, 0x62, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x15, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x0f, 0x22, 0x0a, 0x2f, 0x76,
	0x31, 0x2f, 0x73, 0x75, 0x62, 0x6d, 0x69, 0x74, 0x3a, 0x01, 0x2a, 0x12, 0x46, 0x0a, 0x07, 0x47,
	0x65, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x13, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x47, 0x65, 0x74,
	0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x14, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x47, 0x65, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x10, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x0a, 0x12, 0x08, 0x2f, 0x76, 0x31, 0x2f, 0x69,
	0x6e, 0x66, 0x6f, 0x12, 0x56, 0x0a, 0x08, 0x47, 0x65, 0x74, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x12,
	0x14, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x47, 0x65, 0x74, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x15, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x47, 0x65, 0x74, 0x50,
	0x72, 0x6f, 0x6f, 0x66, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x1d, 0x82, 0xd3,
	0xe4, 0x93, 0x02, 0x17, 0x12, 0x15, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x73,
	0x2f, 0x7b, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x5f, 0x69, 0x64, 0x7d, 0x42, 0x75, 0x0a, 0x07, 0x63,
	0x6f, 0x6d, 0x2e, 0x61, 0x70, 0x69, 0x42, 0x08, 0x41, 0x70, 0x69, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x50, 0x01, 0x5a, 0x34, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x73,
	0x70, 0x61, 0x63, 0x65, 0x6d, 0x65, 0x73, 0x68, 0x6f, 0x73, 0x2f, 0x70, 0x6f, 0x65, 0x74, 0x2f,
	0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f,
	0x2f, 0x72, 0x70, 0x63, 0x2f, 0x61, 0x70, 0x69, 0xa2, 0x02, 0x03, 0x41, 0x58, 0x58, 0xaa, 0x02,
	0x03, 0x41, 0x70, 0x69, 0xca, 0x02, 0x03, 0x41, 0x70, 0x69, 0xe2, 0x02, 0x0f, 0x41, 0x70, 0x69,
	0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x03, 0x41,
	0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_rpc_api_api_proto_rawDescOnce sync.Once
	file_rpc_api_api_proto_rawDescData = file_rpc_api_api_proto_rawDesc
)

func file_rpc_api_api_proto_rawDescGZIP() []byte {
	file_rpc_api_api_proto_rawDescOnce.Do(func() {
		file_rpc_api_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_rpc_api_api_proto_rawDescData)
	})
	return file_rpc_api_api_proto_rawDescData
}

var file_rpc_api_api_proto_msgTypes = make([]protoimpl.MessageInfo, 13)
var file_rpc_api_api_proto_goTypes = []interface{}{
	(*StartRequest)(nil),          // 0: api.StartRequest
	(*StartResponse)(nil),         // 1: api.StartResponse
	(*UpdateGatewayRequest)(nil),  // 2: api.UpdateGatewayRequest
	(*UpdateGatewayResponse)(nil), // 3: api.UpdateGatewayResponse
	(*SubmitRequest)(nil),         // 4: api.SubmitRequest
	(*SubmitResponse)(nil),        // 5: api.SubmitResponse
	(*GetInfoRequest)(nil),        // 6: api.GetInfoRequest
	(*GetInfoResponse)(nil),       // 7: api.GetInfoResponse
	(*MembershipProof)(nil),       // 8: api.MembershipProof
	(*MerkleProof)(nil),           // 9: api.MerkleProof
	(*PoetProof)(nil),             // 10: api.PoetProof
	(*GetProofRequest)(nil),       // 11: api.GetProofRequest
	(*GetProofResponse)(nil),      // 12: api.GetProofResponse
	(*durationpb.Duration)(nil),   // 13: google.protobuf.Duration
}
var file_rpc_api_api_proto_depIdxs = []int32{
	13, // 0: api.SubmitResponse.round_end:type_name -> google.protobuf.Duration
	9,  // 1: api.PoetProof.proof:type_name -> api.MerkleProof
	10, // 2: api.GetProofResponse.proof:type_name -> api.PoetProof
	0,  // 3: api.Poet.Start:input_type -> api.StartRequest
	2,  // 4: api.Poet.UpdateGateway:input_type -> api.UpdateGatewayRequest
	4,  // 5: api.Poet.Submit:input_type -> api.SubmitRequest
	6,  // 6: api.Poet.GetInfo:input_type -> api.GetInfoRequest
	11, // 7: api.Poet.GetProof:input_type -> api.GetProofRequest
	1,  // 8: api.Poet.Start:output_type -> api.StartResponse
	3,  // 9: api.Poet.UpdateGateway:output_type -> api.UpdateGatewayResponse
	5,  // 10: api.Poet.Submit:output_type -> api.SubmitResponse
	7,  // 11: api.Poet.GetInfo:output_type -> api.GetInfoResponse
	12, // 12: api.Poet.GetProof:output_type -> api.GetProofResponse
	8,  // [8:13] is the sub-list for method output_type
	3,  // [3:8] is the sub-list for method input_type
	3,  // [3:3] is the sub-list for extension type_name
	3,  // [3:3] is the sub-list for extension extendee
	0,  // [0:3] is the sub-list for field type_name
}

func init() { file_rpc_api_api_proto_init() }
func file_rpc_api_api_proto_init() {
	if File_rpc_api_api_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_rpc_api_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StartRequest); i {
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
		file_rpc_api_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StartResponse); i {
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
		file_rpc_api_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpdateGatewayRequest); i {
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
		file_rpc_api_api_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpdateGatewayResponse); i {
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
		file_rpc_api_api_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SubmitRequest); i {
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
		file_rpc_api_api_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SubmitResponse); i {
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
		file_rpc_api_api_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetInfoRequest); i {
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
		file_rpc_api_api_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetInfoResponse); i {
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
		file_rpc_api_api_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MembershipProof); i {
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
		file_rpc_api_api_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MerkleProof); i {
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
		file_rpc_api_api_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PoetProof); i {
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
		file_rpc_api_api_proto_msgTypes[11].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetProofRequest); i {
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
		file_rpc_api_api_proto_msgTypes[12].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetProofResponse); i {
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
			RawDescriptor: file_rpc_api_api_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   13,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_rpc_api_api_proto_goTypes,
		DependencyIndexes: file_rpc_api_api_proto_depIdxs,
		MessageInfos:      file_rpc_api_api_proto_msgTypes,
	}.Build()
	File_rpc_api_api_proto = out.File
	file_rpc_api_api_proto_rawDesc = nil
	file_rpc_api_api_proto_goTypes = nil
	file_rpc_api_api_proto_depIdxs = nil
}
