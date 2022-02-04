// Code generated by protoc-gen-go. DO NOT EDIT.
// source: planner.proto

package planner

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type SchedulePlanRequest struct {
	Namespace            string   `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	PodSet               string   `protobuf:"bytes,2,opt,name=podSet,proto3" json:"podSet,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SchedulePlanRequest) Reset()         { *m = SchedulePlanRequest{} }
func (m *SchedulePlanRequest) String() string { return proto.CompactTextString(m) }
func (*SchedulePlanRequest) ProtoMessage()    {}
func (*SchedulePlanRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_3132b275380c0239, []int{0}
}

func (m *SchedulePlanRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SchedulePlanRequest.Unmarshal(m, b)
}
func (m *SchedulePlanRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SchedulePlanRequest.Marshal(b, m, deterministic)
}
func (m *SchedulePlanRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SchedulePlanRequest.Merge(m, src)
}
func (m *SchedulePlanRequest) XXX_Size() int {
	return xxx_messageInfo_SchedulePlanRequest.Size(m)
}
func (m *SchedulePlanRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_SchedulePlanRequest.DiscardUnknown(m)
}

var xxx_messageInfo_SchedulePlanRequest proto.InternalMessageInfo

func (m *SchedulePlanRequest) GetNamespace() string {
	if m != nil {
		return m.Namespace
	}
	return ""
}

func (m *SchedulePlanRequest) GetPodSet() string {
	if m != nil {
		return m.PodSet
	}
	return ""
}

type SchedulePlanResponse struct {
	Assignments          map[string]string `protobuf:"bytes,1,rep,name=assignments,proto3" json:"assignments,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *SchedulePlanResponse) Reset()         { *m = SchedulePlanResponse{} }
func (m *SchedulePlanResponse) String() string { return proto.CompactTextString(m) }
func (*SchedulePlanResponse) ProtoMessage()    {}
func (*SchedulePlanResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_3132b275380c0239, []int{1}
}

func (m *SchedulePlanResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SchedulePlanResponse.Unmarshal(m, b)
}
func (m *SchedulePlanResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SchedulePlanResponse.Marshal(b, m, deterministic)
}
func (m *SchedulePlanResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SchedulePlanResponse.Merge(m, src)
}
func (m *SchedulePlanResponse) XXX_Size() int {
	return xxx_messageInfo_SchedulePlanResponse.Size(m)
}
func (m *SchedulePlanResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_SchedulePlanResponse.DiscardUnknown(m)
}

var xxx_messageInfo_SchedulePlanResponse proto.InternalMessageInfo

func (m *SchedulePlanResponse) GetAssignments() map[string]string {
	if m != nil {
		return m.Assignments
	}
	return nil
}

func init() {
	proto.RegisterType((*SchedulePlanRequest)(nil), "planner.SchedulePlanRequest")
	proto.RegisterType((*SchedulePlanResponse)(nil), "planner.SchedulePlanResponse")
	proto.RegisterMapType((map[string]string)(nil), "planner.SchedulePlanResponse.AssignmentsEntry")
}

func init() { proto.RegisterFile("planner.proto", fileDescriptor_3132b275380c0239) }

var fileDescriptor_3132b275380c0239 = []byte{
	// 229 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2d, 0xc8, 0x49, 0xcc,
	0xcb, 0x4b, 0x2d, 0xd2, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x87, 0x72, 0x95, 0xbc, 0xb9,
	0x84, 0x83, 0x93, 0x33, 0x52, 0x53, 0x4a, 0x73, 0x52, 0x03, 0x72, 0x12, 0xf3, 0x82, 0x52, 0x0b,
	0x4b, 0x53, 0x8b, 0x4b, 0x84, 0x64, 0xb8, 0x38, 0xf3, 0x12, 0x73, 0x53, 0x8b, 0x0b, 0x12, 0x93,
	0x53, 0x25, 0x18, 0x15, 0x18, 0x35, 0x38, 0x83, 0x10, 0x02, 0x42, 0x62, 0x5c, 0x6c, 0x05, 0xf9,
	0x29, 0xc1, 0xa9, 0x25, 0x12, 0x4c, 0x60, 0x29, 0x28, 0x4f, 0x69, 0x05, 0x23, 0x97, 0x08, 0xaa,
	0x69, 0xc5, 0x05, 0xf9, 0x79, 0xc5, 0xa9, 0x42, 0x01, 0x5c, 0xdc, 0x89, 0xc5, 0xc5, 0x99, 0xe9,
	0x79, 0xb9, 0xa9, 0x79, 0x25, 0xc5, 0x12, 0x8c, 0x0a, 0xcc, 0x1a, 0xdc, 0x46, 0x7a, 0x7a, 0x30,
	0x37, 0x61, 0xd3, 0xa3, 0xe7, 0x88, 0xd0, 0xe0, 0x9a, 0x57, 0x52, 0x54, 0x19, 0x84, 0x6c, 0x84,
	0x94, 0x1d, 0x97, 0x00, 0xba, 0x02, 0x21, 0x01, 0x2e, 0xe6, 0xec, 0xd4, 0x4a, 0xa8, 0x73, 0x41,
	0x4c, 0x21, 0x11, 0x2e, 0xd6, 0xb2, 0xc4, 0x9c, 0xd2, 0x54, 0xa8, 0x3b, 0x21, 0x1c, 0x2b, 0x26,
	0x0b, 0x46, 0xa3, 0x64, 0x2e, 0x7e, 0x64, 0x5b, 0xf3, 0x52, 0x8b, 0x84, 0x02, 0xb8, 0x04, 0x9d,
	0x4a, 0x33, 0x73, 0x52, 0x90, 0xc5, 0x85, 0x64, 0x70, 0x38, 0x12, 0x1c, 0x4c, 0x52, 0xb2, 0x78,
	0xbd, 0x90, 0xc4, 0x06, 0x0e, 0x6c, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0x06, 0xbf, 0x4a,
	0x24, 0x7d, 0x01, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// SchedulePlannerClient is the client API for SchedulePlanner service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type SchedulePlannerClient interface {
	BuildSchedulePlan(ctx context.Context, in *SchedulePlanRequest, opts ...grpc.CallOption) (*SchedulePlanResponse, error)
}

type schedulePlannerClient struct {
	cc *grpc.ClientConn
}

func NewSchedulePlannerClient(cc *grpc.ClientConn) SchedulePlannerClient {
	return &schedulePlannerClient{cc}
}

func (c *schedulePlannerClient) BuildSchedulePlan(ctx context.Context, in *SchedulePlanRequest, opts ...grpc.CallOption) (*SchedulePlanResponse, error) {
	out := new(SchedulePlanResponse)
	err := c.cc.Invoke(ctx, "/planner.SchedulePlanner/BuildSchedulePlan", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SchedulePlannerServer is the server API for SchedulePlanner service.
type SchedulePlannerServer interface {
	BuildSchedulePlan(context.Context, *SchedulePlanRequest) (*SchedulePlanResponse, error)
}

// UnimplementedSchedulePlannerServer can be embedded to have forward compatible implementations.
type UnimplementedSchedulePlannerServer struct {
}

func (*UnimplementedSchedulePlannerServer) BuildSchedulePlan(ctx context.Context, req *SchedulePlanRequest) (*SchedulePlanResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BuildSchedulePlan not implemented")
}

func RegisterSchedulePlannerServer(s *grpc.Server, srv SchedulePlannerServer) {
	s.RegisterService(&_SchedulePlanner_serviceDesc, srv)
}

func _SchedulePlanner_BuildSchedulePlan_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SchedulePlanRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SchedulePlannerServer).BuildSchedulePlan(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/planner.SchedulePlanner/BuildSchedulePlan",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SchedulePlannerServer).BuildSchedulePlan(ctx, req.(*SchedulePlanRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _SchedulePlanner_serviceDesc = grpc.ServiceDesc{
	ServiceName: "planner.SchedulePlanner",
	HandlerType: (*SchedulePlannerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "BuildSchedulePlan",
			Handler:    _SchedulePlanner_BuildSchedulePlan_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "planner.proto",
}
