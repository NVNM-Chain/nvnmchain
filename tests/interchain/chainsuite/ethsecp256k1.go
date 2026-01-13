// Package chainsuite contains the ethsecp256k1 key types for EVM-enabled chains.
// This is a minimal copy of the protobuf-generated code from cosmos/evm to avoid
// dependency conflicts with interchaintest.
package chainsuite

import (
	"fmt"
	"io"
	math_bits "math/bits"

	"github.com/cosmos/gogoproto/proto"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.GoGoProtoPackageIsVersion3

// PubKey defines a type alias for an ecdsa.PublicKey that implements
// CometBFT's PubKey interface. It represents the 33-byte compressed public
// key format.
type EthSecp256k1PubKey struct {
	// key is the public key in byte form
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (m *EthSecp256k1PubKey) Reset()      { *m = EthSecp256k1PubKey{} }
func (*EthSecp256k1PubKey) ProtoMessage() {}
func (*EthSecp256k1PubKey) Descriptor() ([]byte, []int) {
	return fileDescriptor_ethsecp256k1, []int{0}
}
func (m *EthSecp256k1PubKey) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *EthSecp256k1PubKey) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_EthSecp256k1PubKey.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *EthSecp256k1PubKey) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EthSecp256k1PubKey.Merge(m, src)
}
func (m *EthSecp256k1PubKey) XXX_Size() int {
	return m.Size()
}
func (m *EthSecp256k1PubKey) XXX_DiscardUnknown() {
	xxx_messageInfo_EthSecp256k1PubKey.DiscardUnknown(m)
}

var xxx_messageInfo_EthSecp256k1PubKey proto.InternalMessageInfo

func (m *EthSecp256k1PubKey) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

// cryptotypes.PubKey interface implementation

func (pubKey EthSecp256k1PubKey) Bytes() []byte {
	return pubKey.Key
}

func (pubKey EthSecp256k1PubKey) String() string {
	return fmt.Sprintf("EthSecp256k1PubKey{%X}", pubKey.Key)
}

func (pubKey EthSecp256k1PubKey) Equals(other cryptotypes.PubKey) bool {
	if other == nil {
		return false
	}
	return string(pubKey.Bytes()) == string(other.Bytes())
}

func (pubKey EthSecp256k1PubKey) Address() cryptotypes.Address {
	if len(pubKey.Key) < 20 {
		return nil
	}
	return pubKey.Key[:20]
}

func (pubKey EthSecp256k1PubKey) Type() string {
	return "eth_secp256k1"
}

func (pubKey EthSecp256k1PubKey) VerifySignature(msg []byte, sig []byte) bool {
	return false // Not needed for transaction decoding
}

// EthSecp256k1PrivKey defines a type alias for an ecdsa.PrivateKey that implements
// CometBFT's PrivateKey interface.
type EthSecp256k1PrivKey struct {
	// key is the private key in byte form
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (m *EthSecp256k1PrivKey) Reset()         { *m = EthSecp256k1PrivKey{} }
func (m *EthSecp256k1PrivKey) String() string { return proto.CompactTextString(m) }
func (*EthSecp256k1PrivKey) ProtoMessage()    {}
func (*EthSecp256k1PrivKey) Descriptor() ([]byte, []int) {
	return fileDescriptor_ethsecp256k1, []int{1}
}
func (m *EthSecp256k1PrivKey) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *EthSecp256k1PrivKey) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_EthSecp256k1PrivKey.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *EthSecp256k1PrivKey) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EthSecp256k1PrivKey.Merge(m, src)
}
func (m *EthSecp256k1PrivKey) XXX_Size() int {
	return m.Size()
}
func (m *EthSecp256k1PrivKey) XXX_DiscardUnknown() {
	xxx_messageInfo_EthSecp256k1PrivKey.DiscardUnknown(m)
}

var xxx_messageInfo_EthSecp256k1PrivKey proto.InternalMessageInfo

func (m *EthSecp256k1PrivKey) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

// cryptotypes.PrivKey interface implementation

func (privKey EthSecp256k1PrivKey) Bytes() []byte {
	return privKey.Key
}

func (privKey EthSecp256k1PrivKey) PubKey() cryptotypes.PubKey {
	return nil // Not needed for transaction decoding
}

func (privKey EthSecp256k1PrivKey) Equals(other cryptotypes.LedgerPrivKey) bool {
	return false
}

func (privKey EthSecp256k1PrivKey) Type() string {
	return "eth_secp256k1"
}

func (privKey EthSecp256k1PrivKey) Sign(msg []byte) ([]byte, error) {
	return nil, nil // Not needed for transaction decoding
}

func init() {
	proto.RegisterType((*EthSecp256k1PubKey)(nil), "cosmos.evm.crypto.v1.ethsecp256k1.PubKey")
	proto.RegisterType((*EthSecp256k1PrivKey)(nil), "cosmos.evm.crypto.v1.ethsecp256k1.PrivKey")
}

func init() {
	proto.RegisterFile("cosmos/evm/crypto/v1/ethsecp256k1/keys.proto", fileDescriptor_ethsecp256k1)
}

var fileDescriptor_ethsecp256k1 = []byte{
	// 193 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xd2, 0x49, 0xce, 0x2f, 0xce,
	0xcd, 0x2f, 0xd6, 0x4f, 0x2d, 0xcb, 0xd5, 0x4f, 0x2e, 0xaa, 0x2c, 0x28, 0xc9, 0xd7, 0x2f, 0x33,
	0xd4, 0x4f, 0x2d, 0xc9, 0x28, 0x4e, 0x4d, 0x2e, 0x30, 0x32, 0x35, 0xcb, 0x36, 0xd4, 0xcf, 0x4e,
	0xad, 0x2c, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x52, 0x84, 0xa8, 0xd6, 0x4b, 0x2d, 0xcb,
	0xd5, 0x83, 0xa8, 0xd6, 0x2b, 0x33, 0xd4, 0x43, 0x56, 0x2d, 0x25, 0x92, 0x9e, 0x9f, 0x9e, 0x0f,
	0x56, 0xad, 0x0f, 0x62, 0x41, 0x34, 0x2a, 0x29, 0x70, 0xb1, 0x05, 0x94, 0x26, 0x79, 0xa7, 0x56,
	0x0a, 0x09, 0x70, 0x31, 0x67, 0xa7, 0x56, 0x4a, 0x30, 0x2a, 0x30, 0x6a, 0xf0, 0x04, 0x81, 0x98,
	0x56, 0x2c, 0x33, 0x16, 0xc8, 0x33, 0x28, 0x49, 0x73, 0xb1, 0x07, 0x14, 0x65, 0x96, 0x61, 0x55,
	0xe2, 0xe4, 0x7c, 0xe2, 0x91, 0x1c, 0xe3, 0x85, 0x47, 0x72, 0x8c, 0x0f, 0x1e, 0xc9, 0x31, 0x4e,
	0x78, 0x2c, 0xc7, 0x70, 0xe1, 0xb1, 0x1c, 0xc3, 0x8d, 0xc7, 0x72, 0x0c, 0x51, 0x9a, 0xe9, 0x99,
	0x25, 0x19, 0xa5, 0x49, 0x7a, 0xc9, 0xf9, 0xb9, 0xfa, 0x98, 0x5e, 0x41, 0x76, 0x59, 0x12, 0x1b,
	0xd8, 0x29, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0x77, 0x7e, 0xb9, 0x2e, 0xf3, 0x00, 0x00,
	0x00,
}

func (m *EthSecp256k1PubKey) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *EthSecp256k1PubKey) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *EthSecp256k1PubKey) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Key) > 0 {
		i -= len(m.Key)
		copy(dAtA[i:], m.Key)
		i = encodeVarintEthsecp256k1(dAtA, i, uint64(len(m.Key)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *EthSecp256k1PrivKey) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *EthSecp256k1PrivKey) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *EthSecp256k1PrivKey) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Key) > 0 {
		i -= len(m.Key)
		copy(dAtA[i:], m.Key)
		i = encodeVarintEthsecp256k1(dAtA, i, uint64(len(m.Key)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintEthsecp256k1(dAtA []byte, offset int, v uint64) int {
	offset -= sovEthsecp256k1(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}

func (m *EthSecp256k1PubKey) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Key)
	if l > 0 {
		n += 1 + l + sovEthsecp256k1(uint64(l))
	}
	return n
}

func (m *EthSecp256k1PrivKey) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Key)
	if l > 0 {
		n += 1 + l + sovEthsecp256k1(uint64(l))
	}
	return n
}

func sovEthsecp256k1(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}

func (m *EthSecp256k1PubKey) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEthsecp256k1
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PubKey: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PubKey: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Key", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEthsecp256k1
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthEthsecp256k1
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEthsecp256k1
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Key = append(m.Key[:0], dAtA[iNdEx:postIndex]...)
			if m.Key == nil {
				m.Key = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEthsecp256k1(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthEthsecp256k1
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func (m *EthSecp256k1PrivKey) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEthsecp256k1
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PrivKey: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PrivKey: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Key", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEthsecp256k1
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthEthsecp256k1
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEthsecp256k1
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Key = append(m.Key[:0], dAtA[iNdEx:postIndex]...)
			if m.Key == nil {
				m.Key = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEthsecp256k1(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthEthsecp256k1
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func skipEthsecp256k1(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowEthsecp256k1
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowEthsecp256k1
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowEthsecp256k1
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthEthsecp256k1
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupEthsecp256k1
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthEthsecp256k1
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthEthsecp256k1        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowEthsecp256k1          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupEthsecp256k1 = fmt.Errorf("proto: unexpected end of group")
)
