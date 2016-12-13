// Code generated by protoc-gen-go.
// source: poe.proto
// DO NOT EDIT!

/*
Package protos is a generated protocol buffer package.

It is generated from these files:
	poe.proto

It has these top-level messages:
	Document
	ProofStat
*/
package protos

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// ProofObject
type Document struct {
	// unique id
	Id string `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	// raw data
	// bytes raw = 2;
	// hash(block)
	BlockDigest string `protobuf:"bytes,3,opt,name=blockDigest" json:"blockDigest,omitempty"`
	// submit time
	SubmitTime int64 `protobuf:"varint,4,opt,name=submitTime" json:"submitTime,omitempty"`
	// proof time
	ProofTime int64 `protobuf:"varint,5,opt,name=proofTime" json:"proofTime,omitempty"`
	// document hash
	Hash string `protobuf:"bytes,6,opt,name=hash" json:"hash,omitempty"`
	// wait time
	WaitDuration int64 `protobuf:"varint,7,opt,name=waitDuration" json:"waitDuration,omitempty"`
	// metadata
	Metadata string `protobuf:"bytes,8,opt,name=metadata" json:"metadata,omitempty"`
	// txid
	Txid string `protobuf:"bytes,9,opt,name=txid" json:"txid,omitempty"`
	// sign
	Sign string `protobuf:"bytes,10,opt,name=sign" json:"sign,omitempty"`
}

func (m *Document) Reset()                    { *m = Document{} }
func (m *Document) String() string            { return proto.CompactTextString(m) }
func (*Document) ProtoMessage()               {}
func (*Document) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// ProofStat
type ProofStat struct {
	StartTime   int64 `protobuf:"varint,1,opt,name=startTime" json:"startTime,omitempty"`
	EndTime     int64 `protobuf:"varint,2,opt,name=endTime" json:"endTime,omitempty"`
	TotalDocs   int64 `protobuf:"varint,3,opt,name=totalDocs" json:"totalDocs,omitempty"`
	WaitDocs    int64 `protobuf:"varint,4,opt,name=waitDocs" json:"waitDocs,omitempty"`
	ProofedDocs int64 `protobuf:"varint,5,opt,name=proofedDocs" json:"proofedDocs,omitempty"`
}

func (m *ProofStat) Reset()                    { *m = ProofStat{} }
func (m *ProofStat) String() string            { return proto.CompactTextString(m) }
func (*ProofStat) ProtoMessage()               {}
func (*ProofStat) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func init() {
	proto.RegisterType((*Document)(nil), "protos.Document")
	proto.RegisterType((*ProofStat)(nil), "protos.ProofStat")
}

func init() { proto.RegisterFile("poe.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 225 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x34, 0x90, 0x41, 0x4e, 0x03, 0x31,
	0x0c, 0x45, 0x35, 0x6d, 0x69, 0x27, 0xa6, 0x12, 0x10, 0x58, 0x64, 0x89, 0xba, 0x62, 0xc5, 0x86,
	0x2b, 0xcc, 0x01, 0x90, 0xe0, 0x02, 0x9e, 0x26, 0xb4, 0x16, 0x4d, 0x5c, 0x25, 0x1e, 0xc1, 0x81,
	0x38, 0x28, 0x89, 0x47, 0xb3, 0x8a, 0xfc, 0x64, 0xfb, 0x3f, 0x07, 0xcc, 0x95, 0xc3, 0xeb, 0x35,
	0xb3, 0xb0, 0xdd, 0xea, 0x53, 0x0e, 0x7f, 0x1d, 0xf4, 0x03, 0x1f, 0xa7, 0x18, 0x92, 0x58, 0x80,
	0x15, 0x79, 0xd7, 0x3d, 0x77, 0x2f, 0xc6, 0x3e, 0xc2, 0xed, 0x78, 0xe1, 0xe3, 0xf7, 0x40, 0xa7,
	0x50, 0xc4, 0xad, 0x15, 0xd6, 0x8e, 0x32, 0x8d, 0x91, 0xe4, 0x93, 0x62, 0x70, 0x9b, 0xca, 0xd6,
	0xf6, 0xa1, 0xae, 0xcd, 0xcc, 0x5f, 0x8a, 0x6e, 0x14, 0xed, 0x61, 0x73, 0xc6, 0x72, 0x76, 0x5b,
	0x1d, 0x7a, 0x82, 0xfd, 0x0f, 0x92, 0x0c, 0x53, 0x46, 0x21, 0x4e, 0x6e, 0xa7, 0x3d, 0xf7, 0xd0,
	0xc7, 0x20, 0xe8, 0x51, 0xd0, 0xf5, 0xda, 0x57, 0xa7, 0xe4, 0xb7, 0xe6, 0x9b, 0xa5, 0x2a, 0x74,
	0x4a, 0x0e, 0x5a, 0x75, 0x20, 0x30, 0xef, 0x2d, 0xe4, 0x43, 0x50, 0x5a, 0x62, 0x11, 0xcc, 0xb3,
	0x44, 0xa7, 0xdb, 0xee, 0x60, 0x17, 0x92, 0x57, 0xb0, 0x5a, 0xac, 0x84, 0x05, 0x2f, 0xf5, 0xb6,
	0xa2, 0xf2, 0x9a, 0xa8, 0x1e, 0x8d, 0xcc, 0xea, 0xf5, 0x46, 0x55, 0x0f, 0x5e, 0xa1, 0xca, 0x8f,
	0xf3, 0xcf, 0xbc, 0xfd, 0x07, 0x00, 0x00, 0xff, 0xff, 0xfe, 0xf6, 0x48, 0x90, 0x2d, 0x01, 0x00,
	0x00,
}
