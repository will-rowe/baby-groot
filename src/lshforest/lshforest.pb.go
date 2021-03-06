// Code generated by protoc-gen-go. DO NOT EDIT.
// source: lshforest.proto

package lshforest

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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

type LSHforest struct {
	K                    int32           `protobuf:"varint,1,opt,name=K,proto3" json:"K,omitempty"`
	L                    int32           `protobuf:"varint,2,opt,name=L,proto3" json:"L,omitempty"`
	KeyLookup            map[string]*Key `protobuf:"bytes,3,rep,name=KeyLookup,proto3" json:"KeyLookup,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Buckets              []*Bucket       `protobuf:"bytes,4,rep,name=Buckets,proto3" json:"Buckets,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *LSHforest) Reset()         { *m = LSHforest{} }
func (m *LSHforest) String() string { return proto.CompactTextString(m) }
func (*LSHforest) ProtoMessage()    {}
func (*LSHforest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a8aa0917749b45ac, []int{0}
}

func (m *LSHforest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LSHforest.Unmarshal(m, b)
}
func (m *LSHforest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LSHforest.Marshal(b, m, deterministic)
}
func (m *LSHforest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LSHforest.Merge(m, src)
}
func (m *LSHforest) XXX_Size() int {
	return xxx_messageInfo_LSHforest.Size(m)
}
func (m *LSHforest) XXX_DiscardUnknown() {
	xxx_messageInfo_LSHforest.DiscardUnknown(m)
}

var xxx_messageInfo_LSHforest proto.InternalMessageInfo

func (m *LSHforest) GetK() int32 {
	if m != nil {
		return m.K
	}
	return 0
}

func (m *LSHforest) GetL() int32 {
	if m != nil {
		return m.L
	}
	return 0
}

func (m *LSHforest) GetKeyLookup() map[string]*Key {
	if m != nil {
		return m.KeyLookup
	}
	return nil
}

func (m *LSHforest) GetBuckets() []*Bucket {
	if m != nil {
		return m.Buckets
	}
	return nil
}

// Bucket is one tree of the LSH Forest
// it is an array of Pairs, where each Pair is a sketch fragment (subsequence) and the graph windows which has that sketch fragment (Key)
type Bucket struct {
	Pairs                []*Pair  `protobuf:"bytes,1,rep,name=Pairs,proto3" json:"Pairs,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Bucket) Reset()         { *m = Bucket{} }
func (m *Bucket) String() string { return proto.CompactTextString(m) }
func (*Bucket) ProtoMessage()    {}
func (*Bucket) Descriptor() ([]byte, []int) {
	return fileDescriptor_a8aa0917749b45ac, []int{1}
}

func (m *Bucket) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Bucket.Unmarshal(m, b)
}
func (m *Bucket) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Bucket.Marshal(b, m, deterministic)
}
func (m *Bucket) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Bucket.Merge(m, src)
}
func (m *Bucket) XXX_Size() int {
	return xxx_messageInfo_Bucket.Size(m)
}
func (m *Bucket) XXX_DiscardUnknown() {
	xxx_messageInfo_Bucket.DiscardUnknown(m)
}

var xxx_messageInfo_Bucket proto.InternalMessageInfo

func (m *Bucket) GetPairs() []*Pair {
	if m != nil {
		return m.Pairs
	}
	return nil
}

type Pair struct {
	SubSequence          string   `protobuf:"bytes,1,opt,name=SubSequence,proto3" json:"SubSequence,omitempty"`
	Keys                 []string `protobuf:"bytes,2,rep,name=Keys,proto3" json:"Keys,omitempty"`
	SketchPartition      []uint64 `protobuf:"varint,3,rep,packed,name=SketchPartition,proto3" json:"SketchPartition,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Pair) Reset()         { *m = Pair{} }
func (m *Pair) String() string { return proto.CompactTextString(m) }
func (*Pair) ProtoMessage()    {}
func (*Pair) Descriptor() ([]byte, []int) {
	return fileDescriptor_a8aa0917749b45ac, []int{2}
}

func (m *Pair) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Pair.Unmarshal(m, b)
}
func (m *Pair) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Pair.Marshal(b, m, deterministic)
}
func (m *Pair) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Pair.Merge(m, src)
}
func (m *Pair) XXX_Size() int {
	return xxx_messageInfo_Pair.Size(m)
}
func (m *Pair) XXX_DiscardUnknown() {
	xxx_messageInfo_Pair.DiscardUnknown(m)
}

var xxx_messageInfo_Pair proto.InternalMessageInfo

func (m *Pair) GetSubSequence() string {
	if m != nil {
		return m.SubSequence
	}
	return ""
}

func (m *Pair) GetKeys() []string {
	if m != nil {
		return m.Keys
	}
	return nil
}

func (m *Pair) GetSketchPartition() []uint64 {
	if m != nil {
		return m.SketchPartition
	}
	return nil
}

// Key relates sketches of reads and graph traversals to specific windows of a graph
type Key struct {
	GraphID              uint32             `protobuf:"varint,1,opt,name=GraphID,proto3" json:"GraphID,omitempty"`
	Node                 uint64             `protobuf:"varint,2,opt,name=Node,proto3" json:"Node,omitempty"`
	OffSet               uint32             `protobuf:"varint,3,opt,name=OffSet,proto3" json:"OffSet,omitempty"`
	ContainedNodes       map[uint64]float64 `protobuf:"bytes,4,rep,name=ContainedNodes,proto3" json:"ContainedNodes,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"fixed64,2,opt,name=value,proto3"`
	Ref                  []uint32           `protobuf:"varint,5,rep,packed,name=Ref,proto3" json:"Ref,omitempty"`
	RC                   bool               `protobuf:"varint,6,opt,name=RC,proto3" json:"RC,omitempty"`
	Sketch               []uint64           `protobuf:"varint,7,rep,packed,name=Sketch,proto3" json:"Sketch,omitempty"`
	Freq                 float64            `protobuf:"fixed64,8,opt,name=Freq,proto3" json:"Freq,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *Key) Reset()         { *m = Key{} }
func (m *Key) String() string { return proto.CompactTextString(m) }
func (*Key) ProtoMessage()    {}
func (*Key) Descriptor() ([]byte, []int) {
	return fileDescriptor_a8aa0917749b45ac, []int{3}
}

func (m *Key) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Key.Unmarshal(m, b)
}
func (m *Key) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Key.Marshal(b, m, deterministic)
}
func (m *Key) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Key.Merge(m, src)
}
func (m *Key) XXX_Size() int {
	return xxx_messageInfo_Key.Size(m)
}
func (m *Key) XXX_DiscardUnknown() {
	xxx_messageInfo_Key.DiscardUnknown(m)
}

var xxx_messageInfo_Key proto.InternalMessageInfo

func (m *Key) GetGraphID() uint32 {
	if m != nil {
		return m.GraphID
	}
	return 0
}

func (m *Key) GetNode() uint64 {
	if m != nil {
		return m.Node
	}
	return 0
}

func (m *Key) GetOffSet() uint32 {
	if m != nil {
		return m.OffSet
	}
	return 0
}

func (m *Key) GetContainedNodes() map[uint64]float64 {
	if m != nil {
		return m.ContainedNodes
	}
	return nil
}

func (m *Key) GetRef() []uint32 {
	if m != nil {
		return m.Ref
	}
	return nil
}

func (m *Key) GetRC() bool {
	if m != nil {
		return m.RC
	}
	return false
}

func (m *Key) GetSketch() []uint64 {
	if m != nil {
		return m.Sketch
	}
	return nil
}

func (m *Key) GetFreq() float64 {
	if m != nil {
		return m.Freq
	}
	return 0
}

func init() {
	proto.RegisterType((*LSHforest)(nil), "lshforest.LSHforest")
	proto.RegisterMapType((map[string]*Key)(nil), "lshforest.LSHforest.KeyLookupEntry")
	proto.RegisterType((*Bucket)(nil), "lshforest.Bucket")
	proto.RegisterType((*Pair)(nil), "lshforest.Pair")
	proto.RegisterType((*Key)(nil), "lshforest.Key")
	proto.RegisterMapType((map[uint64]float64)(nil), "lshforest.Key.ContainedNodesEntry")
}

func init() { proto.RegisterFile("lshforest.proto", fileDescriptor_a8aa0917749b45ac) }

var fileDescriptor_a8aa0917749b45ac = []byte{
	// 405 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x52, 0xdb, 0x6e, 0x9b, 0x40,
	0x10, 0xd5, 0x02, 0xc6, 0x61, 0xdc, 0xe0, 0x76, 0x5b, 0x55, 0xab, 0x3c, 0x21, 0xda, 0x4a, 0x48,
	0x95, 0xa8, 0x94, 0xbe, 0x54, 0x7d, 0x4b, 0xdd, 0x3b, 0xa8, 0x8d, 0x86, 0x2f, 0x20, 0xce, 0x20,
	0x23, 0x22, 0xd6, 0x81, 0xa5, 0x12, 0x7f, 0xd4, 0x1f, 0xeb, 0x7f, 0x54, 0xbb, 0x6b, 0x3b, 0xd8,
	0xca, 0xdb, 0x39, 0xb3, 0x33, 0x67, 0x2e, 0x67, 0x61, 0x79, 0xd7, 0x6f, 0x2a, 0xd9, 0x51, 0xaf,
	0xd2, 0x6d, 0x27, 0x95, 0xe4, 0xc1, 0x21, 0x10, 0xff, 0x63, 0x10, 0xe4, 0xc5, 0x77, 0xcb, 0xf8,
	0x13, 0x60, 0x99, 0x60, 0x11, 0x4b, 0x66, 0xc8, 0x32, 0xcd, 0x72, 0xe1, 0x58, 0x96, 0xf3, 0x2b,
	0x08, 0x32, 0x1a, 0x73, 0x29, 0x9b, 0x61, 0x2b, 0xdc, 0xc8, 0x4d, 0x16, 0x97, 0xaf, 0xd2, 0x07,
	0xe5, 0x83, 0x48, 0x7a, 0xc8, 0xfa, 0xd2, 0xaa, 0x6e, 0xc4, 0x87, 0x2a, 0xfe, 0x16, 0xe6, 0x9f,
	0x86, 0x75, 0x43, 0xaa, 0x17, 0x9e, 0x11, 0x78, 0x36, 0x11, 0xb0, 0x2f, 0xb8, 0xcf, 0xb8, 0xc8,
	0x21, 0x3c, 0x56, 0xe2, 0x4f, 0xc1, 0x6d, 0x68, 0x34, 0xf3, 0x05, 0xa8, 0x21, 0x7f, 0x0d, 0xb3,
	0x3f, 0xe5, 0xdd, 0x40, 0x66, 0xca, 0xc5, 0x65, 0x38, 0x91, 0xcb, 0x68, 0x44, 0xfb, 0xf8, 0xd1,
	0xf9, 0xc0, 0xe2, 0x77, 0xe0, 0x5b, 0x61, 0xfe, 0x06, 0x66, 0xd7, 0x65, 0xdd, 0xf5, 0x82, 0x99,
	0x11, 0x96, 0x93, 0x1a, 0x1d, 0x47, 0xfb, 0x1a, 0x57, 0xe0, 0x69, 0xc0, 0x23, 0x58, 0x14, 0xc3,
	0x4d, 0x41, 0xf7, 0x03, 0xb5, 0x6b, 0xda, 0x35, 0x9f, 0x86, 0x38, 0x07, 0x2f, 0xa3, 0xb1, 0x17,
	0x4e, 0xe4, 0x26, 0x01, 0x1a, 0xcc, 0x13, 0x58, 0x16, 0x0d, 0xa9, 0xf5, 0xe6, 0xba, 0xec, 0x54,
	0xad, 0x6a, 0xd9, 0x9a, 0x93, 0x79, 0x78, 0x1a, 0x8e, 0xff, 0x3a, 0xe0, 0x66, 0x34, 0x72, 0x01,
	0xf3, 0x6f, 0x5d, 0xb9, 0xdd, 0xfc, 0xf8, 0x6c, 0x7a, 0x9c, 0xe3, 0x9e, 0x6a, 0xfd, 0x5f, 0xf2,
	0xd6, 0xee, 0xe8, 0xa1, 0xc1, 0xfc, 0x25, 0xf8, 0xbf, 0xab, 0xaa, 0x20, 0x25, 0x5c, 0x93, 0xbc,
	0x63, 0xfc, 0x27, 0x84, 0x2b, 0xd9, 0xaa, 0xb2, 0x6e, 0xe9, 0x56, 0x27, 0xee, 0x0f, 0x1d, 0x1f,
	0x5f, 0x26, 0x3d, 0x4e, 0xb2, 0x46, 0x9d, 0x54, 0xea, 0x73, 0x23, 0x55, 0x62, 0x16, 0xb9, 0xc9,
	0x39, 0x6a, 0xc8, 0x43, 0x70, 0x70, 0x25, 0xfc, 0x88, 0x25, 0x67, 0xe8, 0xe0, 0x4a, 0x4f, 0x61,
	0xd7, 0x11, 0x73, 0xb3, 0xdc, 0x8e, 0xe9, 0x89, 0xbf, 0x76, 0x74, 0x2f, 0xce, 0x22, 0x96, 0x30,
	0x34, 0xf8, 0xe2, 0x0a, 0x9e, 0x3f, 0xd2, 0x74, 0xea, 0xa9, 0x67, 0x3d, 0x7d, 0x31, 0xf5, 0x94,
	0x4d, 0x3c, 0xbc, 0xf1, 0xcd, 0xef, 0x7d, 0xff, 0x3f, 0x00, 0x00, 0xff, 0xff, 0x51, 0x35, 0x35,
	0x59, 0xd0, 0x02, 0x00, 0x00,
}
