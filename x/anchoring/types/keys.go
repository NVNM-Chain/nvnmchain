package types

import (
	"cosmossdk.io/collections"
)

const (
	// ModuleName defines the module name
	ModuleName = "anchoring"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

var (
	ParamsKey                              = collections.NewPrefix(0)
	RegistriesKeyPrefix                    = collections.NewPrefix(1)
	RegistryIdByNameKeyPrefix              = collections.NewPrefix(2)
	RegistryCountKey                       = collections.NewPrefix(3)
	RecordsKeyPrefix                       = collections.NewPrefix(4)
	RecordIdByRegistryAndChecksumKeyPrefix = collections.NewPrefix(5)
	RecordsCountByRegistryIdKeyPrefix      = collections.NewPrefix(6)
	RecordIndicesKeyPrefix                 = collections.NewPrefix(7)
	RecordIdByChecksumAndRegistryKeyPrefix = collections.NewPrefix(8)
	RolesKeyPrefix                         = collections.NewPrefix(9)
	RoleAdminsKeyPrefix                    = collections.NewPrefix(10)
	RoleMembersKeyPrefix                   = collections.NewPrefix(11)
)
