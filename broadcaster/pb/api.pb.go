// Code generated by protoc-gen-go. DO NOT EDIT.
// source: broadcaster/pb/api.proto

package pb

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type SimpleMessage struct {
	Value                string   `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SimpleMessage) Reset()         { *m = SimpleMessage{} }
func (m *SimpleMessage) String() string { return proto.CompactTextString(m) }
func (*SimpleMessage) ProtoMessage()    {}
func (*SimpleMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_8065f5ea669012df, []int{0}
}

func (m *SimpleMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SimpleMessage.Unmarshal(m, b)
}
func (m *SimpleMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SimpleMessage.Marshal(b, m, deterministic)
}
func (m *SimpleMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SimpleMessage.Merge(m, src)
}
func (m *SimpleMessage) XXX_Size() int {
	return xxx_messageInfo_SimpleMessage.Size(m)
}
func (m *SimpleMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_SimpleMessage.DiscardUnknown(m)
}

var xxx_messageInfo_SimpleMessage proto.InternalMessageInfo

func (m *SimpleMessage) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

type AccountId struct {
	Address              string   `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AccountId) Reset()         { *m = AccountId{} }
func (m *AccountId) String() string { return proto.CompactTextString(m) }
func (*AccountId) ProtoMessage()    {}
func (*AccountId) Descriptor() ([]byte, []int) {
	return fileDescriptor_8065f5ea669012df, []int{1}
}

func (m *AccountId) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AccountId.Unmarshal(m, b)
}
func (m *AccountId) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AccountId.Marshal(b, m, deterministic)
}
func (m *AccountId) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AccountId.Merge(m, src)
}
func (m *AccountId) XXX_Size() int {
	return xxx_messageInfo_AccountId.Size(m)
}
func (m *AccountId) XXX_DiscardUnknown() {
	xxx_messageInfo_AccountId.DiscardUnknown(m)
}

var xxx_messageInfo_AccountId proto.InternalMessageInfo

func (m *AccountId) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

type TransferFunds struct {
	Sender               *AccountId `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
	Receiver             *AccountId `protobuf:"bytes,2,opt,name=receiver,proto3" json:"receiver,omitempty"`
	Nonce                uint64     `protobuf:"varint,3,opt,name=nonce,proto3" json:"nonce,omitempty"`
	Amount               uint64     `protobuf:"varint,4,opt,name=amount,proto3" json:"amount,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *TransferFunds) Reset()         { *m = TransferFunds{} }
func (m *TransferFunds) String() string { return proto.CompactTextString(m) }
func (*TransferFunds) ProtoMessage()    {}
func (*TransferFunds) Descriptor() ([]byte, []int) {
	return fileDescriptor_8065f5ea669012df, []int{2}
}

func (m *TransferFunds) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TransferFunds.Unmarshal(m, b)
}
func (m *TransferFunds) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TransferFunds.Marshal(b, m, deterministic)
}
func (m *TransferFunds) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TransferFunds.Merge(m, src)
}
func (m *TransferFunds) XXX_Size() int {
	return xxx_messageInfo_TransferFunds.Size(m)
}
func (m *TransferFunds) XXX_DiscardUnknown() {
	xxx_messageInfo_TransferFunds.DiscardUnknown(m)
}

var xxx_messageInfo_TransferFunds proto.InternalMessageInfo

func (m *TransferFunds) GetSender() *AccountId {
	if m != nil {
		return m.Sender
	}
	return nil
}

func (m *TransferFunds) GetReceiver() *AccountId {
	if m != nil {
		return m.Receiver
	}
	return nil
}

func (m *TransferFunds) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *TransferFunds) GetAmount() uint64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

type SignedTransaction struct {
	SrcAddress           string   `protobuf:"bytes,1,opt,name=srcAddress,proto3" json:"srcAddress,omitempty"`
	DstAddress           string   `protobuf:"bytes,2,opt,name=dstAddress,proto3" json:"dstAddress,omitempty"`
	Amount               string   `protobuf:"bytes,3,opt,name=amount,proto3" json:"amount,omitempty"`
	Nonce                string   `protobuf:"bytes,4,opt,name=nonce,proto3" json:"nonce,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SignedTransaction) Reset()         { *m = SignedTransaction{} }
func (m *SignedTransaction) String() string { return proto.CompactTextString(m) }
func (*SignedTransaction) ProtoMessage()    {}
func (*SignedTransaction) Descriptor() ([]byte, []int) {
	return fileDescriptor_8065f5ea669012df, []int{3}
}

func (m *SignedTransaction) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SignedTransaction.Unmarshal(m, b)
}
func (m *SignedTransaction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SignedTransaction.Marshal(b, m, deterministic)
}
func (m *SignedTransaction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SignedTransaction.Merge(m, src)
}
func (m *SignedTransaction) XXX_Size() int {
	return xxx_messageInfo_SignedTransaction.Size(m)
}
func (m *SignedTransaction) XXX_DiscardUnknown() {
	xxx_messageInfo_SignedTransaction.DiscardUnknown(m)
}

var xxx_messageInfo_SignedTransaction proto.InternalMessageInfo

func (m *SignedTransaction) GetSrcAddress() string {
	if m != nil {
		return m.SrcAddress
	}
	return ""
}

func (m *SignedTransaction) GetDstAddress() string {
	if m != nil {
		return m.DstAddress
	}
	return ""
}

func (m *SignedTransaction) GetAmount() string {
	if m != nil {
		return m.Amount
	}
	return ""
}

func (m *SignedTransaction) GetNonce() string {
	if m != nil {
		return m.Nonce
	}
	return ""
}

type BroadcastMessage struct {
	Data                 string   `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BroadcastMessage) Reset()         { *m = BroadcastMessage{} }
func (m *BroadcastMessage) String() string { return proto.CompactTextString(m) }
func (*BroadcastMessage) ProtoMessage()    {}
func (*BroadcastMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_8065f5ea669012df, []int{4}
}

func (m *BroadcastMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BroadcastMessage.Unmarshal(m, b)
}
func (m *BroadcastMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BroadcastMessage.Marshal(b, m, deterministic)
}
func (m *BroadcastMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BroadcastMessage.Merge(m, src)
}
func (m *BroadcastMessage) XXX_Size() int {
	return xxx_messageInfo_BroadcastMessage.Size(m)
}
func (m *BroadcastMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_BroadcastMessage.DiscardUnknown(m)
}

var xxx_messageInfo_BroadcastMessage proto.InternalMessageInfo

func (m *BroadcastMessage) GetData() string {
	if m != nil {
		return m.Data
	}
	return ""
}

type BinaryMessage struct {
	Data                 []byte   `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BinaryMessage) Reset()         { *m = BinaryMessage{} }
func (m *BinaryMessage) String() string { return proto.CompactTextString(m) }
func (*BinaryMessage) ProtoMessage()    {}
func (*BinaryMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_8065f5ea669012df, []int{5}
}

func (m *BinaryMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BinaryMessage.Unmarshal(m, b)
}
func (m *BinaryMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BinaryMessage.Marshal(b, m, deterministic)
}
func (m *BinaryMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BinaryMessage.Merge(m, src)
}
func (m *BinaryMessage) XXX_Size() int {
	return xxx_messageInfo_BinaryMessage.Size(m)
}
func (m *BinaryMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_BinaryMessage.DiscardUnknown(m)
}

var xxx_messageInfo_BinaryMessage proto.InternalMessageInfo

func (m *BinaryMessage) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

type CommitmentSizeMessage struct {
	MbCommitted          uint64   `protobuf:"varint,1,opt,name=mbCommitted,proto3" json:"mbCommitted,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CommitmentSizeMessage) Reset()         { *m = CommitmentSizeMessage{} }
func (m *CommitmentSizeMessage) String() string { return proto.CompactTextString(m) }
func (*CommitmentSizeMessage) ProtoMessage()    {}
func (*CommitmentSizeMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_8065f5ea669012df, []int{6}
}

func (m *CommitmentSizeMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CommitmentSizeMessage.Unmarshal(m, b)
}
func (m *CommitmentSizeMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CommitmentSizeMessage.Marshal(b, m, deterministic)
}
func (m *CommitmentSizeMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CommitmentSizeMessage.Merge(m, src)
}
func (m *CommitmentSizeMessage) XXX_Size() int {
	return xxx_messageInfo_CommitmentSizeMessage.Size(m)
}
func (m *CommitmentSizeMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_CommitmentSizeMessage.DiscardUnknown(m)
}

var xxx_messageInfo_CommitmentSizeMessage proto.InternalMessageInfo

func (m *CommitmentSizeMessage) GetMbCommitted() uint64 {
	if m != nil {
		return m.MbCommitted
	}
	return 0
}

type LogicalDriveMessage struct {
	LogicalDrive         string   `protobuf:"bytes,1,opt,name=logicalDrive,proto3" json:"logicalDrive,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *LogicalDriveMessage) Reset()         { *m = LogicalDriveMessage{} }
func (m *LogicalDriveMessage) String() string { return proto.CompactTextString(m) }
func (*LogicalDriveMessage) ProtoMessage()    {}
func (*LogicalDriveMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_8065f5ea669012df, []int{7}
}

func (m *LogicalDriveMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LogicalDriveMessage.Unmarshal(m, b)
}
func (m *LogicalDriveMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LogicalDriveMessage.Marshal(b, m, deterministic)
}
func (m *LogicalDriveMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogicalDriveMessage.Merge(m, src)
}
func (m *LogicalDriveMessage) XXX_Size() int {
	return xxx_messageInfo_LogicalDriveMessage.Size(m)
}
func (m *LogicalDriveMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_LogicalDriveMessage.DiscardUnknown(m)
}

var xxx_messageInfo_LogicalDriveMessage proto.InternalMessageInfo

func (m *LogicalDriveMessage) GetLogicalDrive() string {
	if m != nil {
		return m.LogicalDrive
	}
	return ""
}

func init() {
	proto.RegisterType((*SimpleMessage)(nil), "pb.SimpleMessage")
	proto.RegisterType((*AccountId)(nil), "pb.AccountId")
	proto.RegisterType((*TransferFunds)(nil), "pb.TransferFunds")
	proto.RegisterType((*SignedTransaction)(nil), "pb.SignedTransaction")
	proto.RegisterType((*BroadcastMessage)(nil), "pb.BroadcastMessage")
	proto.RegisterType((*BinaryMessage)(nil), "pb.BinaryMessage")
	proto.RegisterType((*CommitmentSizeMessage)(nil), "pb.CommitmentSizeMessage")
	proto.RegisterType((*LogicalDriveMessage)(nil), "pb.LogicalDriveMessage")
}

func init() { proto.RegisterFile("broadcaster/pb/api.proto", fileDescriptor_8065f5ea669012df) }

var fileDescriptor_8065f5ea669012df = []byte{
	// 729 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x94, 0xdd, 0x4e, 0xdb, 0x48,
	0x14, 0xc7, 0x95, 0x90, 0x65, 0xc9, 0x81, 0x40, 0x32, 0x24, 0xe0, 0x0d, 0x2c, 0x8a, 0x66, 0xc5,
	0x8a, 0xdd, 0x8b, 0x44, 0xcb, 0x5e, 0xd1, 0x3b, 0xd2, 0x42, 0x84, 0xd4, 0x22, 0x14, 0x53, 0xa9,
	0xaa, 0x54, 0xb5, 0x63, 0xfb, 0x60, 0x46, 0x8a, 0x3d, 0xd6, 0xcc, 0x24, 0x2d, 0xdc, 0xb5, 0x2f,
	0xd0, 0x8b, 0x3e, 0x40, 0x1f, 0xaa, 0xaf, 0xd0, 0x07, 0xa9, 0x66, 0xec, 0x98, 0x98, 0x5a, 0xa2,
	0xdc, 0xe5, 0x7c, 0xfd, 0xce, 0xc9, 0x39, 0x7f, 0x0f, 0x38, 0x9e, 0x14, 0x2c, 0xf0, 0x99, 0xd2,
	0x28, 0x07, 0x89, 0x37, 0x60, 0x09, 0xef, 0x27, 0x52, 0x68, 0x41, 0xaa, 0x89, 0xd7, 0xdd, 0x0d,
	0x85, 0x08, 0x27, 0x68, 0xbc, 0x03, 0x16, 0xc7, 0x42, 0x33, 0xcd, 0x45, 0xac, 0xd2, 0x8c, 0xee,
	0x4e, 0x16, 0xb5, 0x96, 0x37, 0xbd, 0x1a, 0x60, 0x94, 0xe8, 0x9b, 0x34, 0x48, 0xf7, 0xa1, 0xe1,
	0xf2, 0x28, 0x99, 0xe0, 0x0b, 0x54, 0x8a, 0x85, 0x48, 0xda, 0xf0, 0xdb, 0x8c, 0x4d, 0xa6, 0xe8,
	0x54, 0x7a, 0x95, 0x83, 0xfa, 0x38, 0x35, 0xe8, 0x3e, 0xd4, 0x8f, 0x7d, 0x5f, 0x4c, 0x63, 0x7d,
	0x16, 0x10, 0x07, 0x7e, 0x67, 0x41, 0x20, 0x51, 0xa9, 0x2c, 0x69, 0x6e, 0xd2, 0xcf, 0x15, 0x68,
	0x5c, 0x4a, 0x16, 0xab, 0x2b, 0x94, 0xa7, 0xd3, 0x38, 0x50, 0x64, 0x1f, 0x96, 0x15, 0xc6, 0x01,
	0x4a, 0x9b, 0xba, 0x7a, 0xd8, 0xe8, 0x27, 0x5e, 0x3f, 0x47, 0x8d, 0xb3, 0x20, 0xf9, 0x07, 0x56,
	0x24, 0xfa, 0xc8, 0x67, 0x28, 0x9d, 0x6a, 0x59, 0x62, 0x1e, 0x36, 0x03, 0xc6, 0x22, 0xf6, 0xd1,
	0x59, 0xea, 0x55, 0x0e, 0x6a, 0xe3, 0xd4, 0x20, 0x5b, 0xb0, 0xcc, 0x22, 0x93, 0xeb, 0xd4, 0xac,
	0x3b, 0xb3, 0xe8, 0xc7, 0x0a, 0xb4, 0x5c, 0x1e, 0xc6, 0x18, 0xd8, 0xb9, 0x98, 0x6f, 0x36, 0x43,
	0xf6, 0x00, 0x94, 0xf4, 0x8f, 0x0b, 0x7f, 0x62, 0xc1, 0x63, 0xe2, 0x81, 0xd2, 0xf3, 0x78, 0x35,
	0x8d, 0xdf, 0x79, 0x16, 0xba, 0x2d, 0xd9, 0x58, 0x66, 0xdd, 0xcd, 0x56, 0x4b, 0x97, 0x67, 0x0d,
	0xfa, 0x37, 0x34, 0x87, 0xf3, 0xf3, 0xcd, 0xd7, 0x4c, 0xa0, 0x16, 0x30, 0xcd, 0xb2, 0xde, 0xf6,
	0x37, 0xfd, 0x0b, 0x1a, 0x43, 0x1e, 0x33, 0x79, 0x53, 0x96, 0xb4, 0x96, 0x25, 0x1d, 0x41, 0xe7,
	0xa9, 0x88, 0x22, 0xae, 0x23, 0x8c, 0xb5, 0xcb, 0x6f, 0xf3, 0xc3, 0xf5, 0x60, 0x35, 0xf2, 0xd2,
	0x90, 0xc6, 0xc0, 0xd6, 0xd4, 0xc6, 0x8b, 0x2e, 0x7a, 0x04, 0x9b, 0xcf, 0x45, 0xc8, 0x7d, 0x36,
	0x79, 0x26, 0xf9, 0x2c, 0x2f, 0xa4, 0xb0, 0x36, 0x59, 0x70, 0x67, 0x23, 0x15, 0x7c, 0x87, 0x5f,
	0x57, 0xa0, 0xe9, 0x26, 0xcc, 0xc7, 0x08, 0xd5, 0xb5, 0x8b, 0x72, 0xc6, 0x7d, 0x24, 0x67, 0x50,
	0x3b, 0xf1, 0xaf, 0x05, 0x69, 0x99, 0x53, 0x15, 0x54, 0xd4, 0xfd, 0xd9, 0x45, 0x77, 0x3e, 0x7d,
	0xfb, 0xfe, 0xa5, 0xda, 0xa1, 0xcd, 0xc1, 0xec, 0xbf, 0x01, 0x7e, 0x60, 0x26, 0x36, 0x40, 0xff,
	0x5a, 0x3c, 0xa9, 0xfc, 0x4b, 0x86, 0xb0, 0x32, 0x42, 0x7d, 0x6e, 0x4f, 0x59, 0xbc, 0x7c, 0x19,
	0xaa, 0x6d, 0x51, 0xeb, 0xb4, 0x6e, 0x50, 0x76, 0xc7, 0x86, 0x71, 0x0a, 0x30, 0x42, 0x3d, 0x64,
	0x13, 0xf6, 0x6b, 0x94, 0x2d, 0x4b, 0x69, 0xd2, 0x55, 0x43, 0xf1, 0xd2, 0x32, 0xc3, 0x79, 0x0b,
	0x2d, 0x77, 0xea, 0x45, 0x5c, 0x2f, 0x2a, 0xa6, 0x93, 0xd6, 0xdf, 0x13, 0x52, 0x19, 0xb6, 0x67,
	0xb1, 0x5d, 0xda, 0x31, 0x58, 0x65, 0x41, 0xfa, 0xae, 0xc2, 0x34, 0x38, 0x87, 0x7a, 0xae, 0x07,
	0xd2, 0x36, 0x84, 0xfb, 0xf2, 0x28, 0xe3, 0x3a, 0x96, 0x4b, 0x68, 0xc3, 0x8e, 0x3b, 0x2f, 0x30,
	0x3c, 0x17, 0x1a, 0x39, 0xe0, 0x42, 0xa0, 0x4e, 0x0f, 0x52, 0x90, 0x52, 0x19, 0x70, 0xd7, 0x02,
	0xb7, 0x68, 0xab, 0x00, 0x4c, 0x04, 0x5a, 0xa8, 0x0f, 0x2d, 0x17, 0x75, 0x51, 0x6a, 0xe4, 0x0f,
	0x43, 0x29, 0x95, 0xdf, 0xc3, 0x9b, 0x40, 0xed, 0xe7, 0x85, 0x8a, 0xdf, 0xda, 0x55, 0xbf, 0x81,
	0x0d, 0x17, 0xf5, 0xa2, 0x28, 0xc9, 0xb6, 0xe1, 0x94, 0xc8, 0xb4, 0xac, 0xc1, 0x9e, 0x6d, 0xe0,
	0xd0, 0xcd, 0xac, 0x41, 0x26, 0xd9, 0xc0, 0x94, 0x19, 0xfc, 0x18, 0x9a, 0x2e, 0xea, 0xe3, 0xf7,
	0x4c, 0x06, 0x6a, 0xfe, 0xe9, 0x3e, 0xac, 0x8b, 0xc2, 0x5e, 0x14, 0x6a, 0x66, 0xeb, 0xcd, 0x13,
	0x67, 0x98, 0xaf, 0x60, 0x63, 0x84, 0xfa, 0x2c, 0xe6, 0xfa, 0x42, 0x8a, 0x30, 0x7d, 0x0d, 0xfa,
	0xe9, 0x0b, 0xdb, 0x9f, 0xbf, 0xb0, 0xfd, 0x13, 0xf3, 0xc2, 0x96, 0xb1, 0xbb, 0x96, 0xdd, 0xa6,
	0x1b, 0x86, 0x1d, 0xa2, 0x4e, 0x32, 0x86, 0x21, 0xbf, 0x86, 0xf5, 0x11, 0xea, 0x4b, 0xa1, 0xd9,
	0x24, 0x1d, 0xf9, 0x31, 0xe0, 0x3f, 0x2d, 0x78, 0x9b, 0x92, 0x0c, 0xac, 0x0d, 0x26, 0x9d, 0xdc,
	0xb0, 0xdf, 0x41, 0x6b, 0x84, 0xfa, 0x65, 0xe2, 0x8b, 0x88, 0xc7, 0xe1, 0xe3, 0xf1, 0x85, 0x53,
	0x86, 0xa8, 0xa7, 0x19, 0x29, 0xef, 0xe0, 0x2d, 0x5b, 0xc8, 0xff, 0x3f, 0x02, 0x00, 0x00, 0xff,
	0xff, 0x69, 0x1c, 0xd7, 0xd3, 0xaa, 0x06, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// SpacemeshServiceClient is the client API for SpacemeshService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type SpacemeshServiceClient interface {
	Echo(ctx context.Context, in *SimpleMessage, opts ...grpc.CallOption) (*SimpleMessage, error)
	GetNonce(ctx context.Context, in *AccountId, opts ...grpc.CallOption) (*SimpleMessage, error)
	GetBalance(ctx context.Context, in *AccountId, opts ...grpc.CallOption) (*SimpleMessage, error)
	SubmitTransaction(ctx context.Context, in *SignedTransaction, opts ...grpc.CallOption) (*SimpleMessage, error)
	Broadcast(ctx context.Context, in *BroadcastMessage, opts ...grpc.CallOption) (*SimpleMessage, error)
	BroadcastPoet(ctx context.Context, in *BinaryMessage, opts ...grpc.CallOption) (*SimpleMessage, error)
	SetCommitmentSize(ctx context.Context, in *CommitmentSizeMessage, opts ...grpc.CallOption) (*SimpleMessage, error)
	SetLogicalDrive(ctx context.Context, in *LogicalDriveMessage, opts ...grpc.CallOption) (*SimpleMessage, error)
	SetAwardsAddress(ctx context.Context, in *AccountId, opts ...grpc.CallOption) (*SimpleMessage, error)
	GetInitProgress(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*SimpleMessage, error)
	GetTotalAwards(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*SimpleMessage, error)
	GetUpcomingAwards(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*SimpleMessage, error)
}

type spacemeshServiceClient struct {
	cc *grpc.ClientConn
}

func NewSpacemeshServiceClient(cc *grpc.ClientConn) SpacemeshServiceClient {
	return &spacemeshServiceClient{cc}
}

func (c *spacemeshServiceClient) Echo(ctx context.Context, in *SimpleMessage, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/Echo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) GetNonce(ctx context.Context, in *AccountId, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/GetNonce", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) GetBalance(ctx context.Context, in *AccountId, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/GetBalance", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) SubmitTransaction(ctx context.Context, in *SignedTransaction, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/SubmitTransaction", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) Broadcast(ctx context.Context, in *BroadcastMessage, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/Broadcast", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) BroadcastPoet(ctx context.Context, in *BinaryMessage, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/BroadcastPoet", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) SetCommitmentSize(ctx context.Context, in *CommitmentSizeMessage, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/SetCommitmentSize", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) SetLogicalDrive(ctx context.Context, in *LogicalDriveMessage, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/SetLogicalDrive", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) SetAwardsAddress(ctx context.Context, in *AccountId, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/SetAwardsAddress", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) GetInitProgress(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/GetInitProgress", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) GetTotalAwards(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/GetTotalAwards", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spacemeshServiceClient) GetUpcomingAwards(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*SimpleMessage, error) {
	out := new(SimpleMessage)
	err := c.cc.Invoke(ctx, "/pb.SpacemeshService/GetUpcomingAwards", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SpacemeshServiceServer is the server API for SpacemeshService service.
type SpacemeshServiceServer interface {
	Echo(context.Context, *SimpleMessage) (*SimpleMessage, error)
	GetNonce(context.Context, *AccountId) (*SimpleMessage, error)
	GetBalance(context.Context, *AccountId) (*SimpleMessage, error)
	SubmitTransaction(context.Context, *SignedTransaction) (*SimpleMessage, error)
	Broadcast(context.Context, *BroadcastMessage) (*SimpleMessage, error)
	BroadcastPoet(context.Context, *BinaryMessage) (*SimpleMessage, error)
	SetCommitmentSize(context.Context, *CommitmentSizeMessage) (*SimpleMessage, error)
	SetLogicalDrive(context.Context, *LogicalDriveMessage) (*SimpleMessage, error)
	SetAwardsAddress(context.Context, *AccountId) (*SimpleMessage, error)
	GetInitProgress(context.Context, *empty.Empty) (*SimpleMessage, error)
	GetTotalAwards(context.Context, *empty.Empty) (*SimpleMessage, error)
	GetUpcomingAwards(context.Context, *empty.Empty) (*SimpleMessage, error)
}

func RegisterSpacemeshServiceServer(s *grpc.Server, srv SpacemeshServiceServer) {
	s.RegisterService(&_SpacemeshService_serviceDesc, srv)
}

func _SpacemeshService_Echo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SimpleMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).Echo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/Echo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).Echo(ctx, req.(*SimpleMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_GetNonce_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AccountId)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).GetNonce(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/GetNonce",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).GetNonce(ctx, req.(*AccountId))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_GetBalance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AccountId)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).GetBalance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).GetBalance(ctx, req.(*AccountId))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_SubmitTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignedTransaction)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).SubmitTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/SubmitTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).SubmitTransaction(ctx, req.(*SignedTransaction))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_Broadcast_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BroadcastMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).Broadcast(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/Broadcast",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).Broadcast(ctx, req.(*BroadcastMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_BroadcastPoet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BinaryMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).BroadcastPoet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/BroadcastPoet",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).BroadcastPoet(ctx, req.(*BinaryMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_SetCommitmentSize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommitmentSizeMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).SetCommitmentSize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/SetCommitmentSize",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).SetCommitmentSize(ctx, req.(*CommitmentSizeMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_SetLogicalDrive_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LogicalDriveMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).SetLogicalDrive(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/SetLogicalDrive",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).SetLogicalDrive(ctx, req.(*LogicalDriveMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_SetAwardsAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AccountId)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).SetAwardsAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/SetAwardsAddress",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).SetAwardsAddress(ctx, req.(*AccountId))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_GetInitProgress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).GetInitProgress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/GetInitProgress",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).GetInitProgress(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_GetTotalAwards_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).GetTotalAwards(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/GetTotalAwards",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).GetTotalAwards(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpacemeshService_GetUpcomingAwards_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpacemeshServiceServer).GetUpcomingAwards(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SpacemeshService/GetUpcomingAwards",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpacemeshServiceServer).GetUpcomingAwards(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

var _SpacemeshService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pb.SpacemeshService",
	HandlerType: (*SpacemeshServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Echo",
			Handler:    _SpacemeshService_Echo_Handler,
		},
		{
			MethodName: "GetNonce",
			Handler:    _SpacemeshService_GetNonce_Handler,
		},
		{
			MethodName: "GetBalance",
			Handler:    _SpacemeshService_GetBalance_Handler,
		},
		{
			MethodName: "SubmitTransaction",
			Handler:    _SpacemeshService_SubmitTransaction_Handler,
		},
		{
			MethodName: "Broadcast",
			Handler:    _SpacemeshService_Broadcast_Handler,
		},
		{
			MethodName: "BroadcastPoet",
			Handler:    _SpacemeshService_BroadcastPoet_Handler,
		},
		{
			MethodName: "SetCommitmentSize",
			Handler:    _SpacemeshService_SetCommitmentSize_Handler,
		},
		{
			MethodName: "SetLogicalDrive",
			Handler:    _SpacemeshService_SetLogicalDrive_Handler,
		},
		{
			MethodName: "SetAwardsAddress",
			Handler:    _SpacemeshService_SetAwardsAddress_Handler,
		},
		{
			MethodName: "GetInitProgress",
			Handler:    _SpacemeshService_GetInitProgress_Handler,
		},
		{
			MethodName: "GetTotalAwards",
			Handler:    _SpacemeshService_GetTotalAwards_Handler,
		},
		{
			MethodName: "GetUpcomingAwards",
			Handler:    _SpacemeshService_GetUpcomingAwards_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "broadcaster/pb/api.proto",
}