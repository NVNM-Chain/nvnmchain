package rbac

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/codec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Role common.Hash

func NewRole(bz []byte) Role {
	return Role(common.BytesToHash(bz))
}

func NewRoleHash(id string) Role {
	return Role(crypto.Keccak256Hash([]byte(id)))
}

func (role Role) Bytes() []byte {
	return common.Hash(role).Bytes()
}

func (role Role) String() string {
	return common.Hash(role).Hex()
}

type RoleCodec struct{}

var _ codec.KeyCodec[Role] = RoleCodec{}

func (RoleCodec) Encode(buffer []byte, key Role) (int, error) {
	return collections.BytesKey.Encode(buffer, key.Bytes())
}

func (RoleCodec) Decode(buffer []byte) (int, Role, error) {
	n, bz, err := collections.BytesKey.Decode(buffer)
	if err != nil {
		return n, Role{}, err
	}
	return n, NewRole(bz), nil
}

func (RoleCodec) Size(key Role) int {
	return 32
}

func (RoleCodec) EncodeJSON(value Role) ([]byte, error) {
	return collections.StringKey.EncodeJSON(value.String())
}

func (RoleCodec) DecodeJSON(b []byte) (Role, error) {
	s, err := collections.StringKey.DecodeJSON(b)
	if err != nil {
		return Role{}, err
	}
	return NewRole(common.FromHex(s)), nil
}

func (RoleCodec) Stringify(key Role) string {
	return key.String()
}

func (RoleCodec) KeyType() string {
	return "rbac.Role"
}

func (RoleCodec) ValueType() string {
	return "rbac.Role"
}

func (k RoleCodec) EncodeNonTerminal(buffer []byte, key Role) (int, error) {
	return k.Encode(buffer, key)
}

func (k RoleCodec) DecodeNonTerminal(buffer []byte) (int, Role, error) {
	return k.Decode(buffer)
}

func (RoleCodec) SizeNonTerminal(key Role) int {
	return 32
}
