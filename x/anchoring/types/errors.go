package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/anchoring module sentinel errors
var (
	ErrRegistryExists                        = sdkerrors.Register(ModuleName, 2100, "registry already exists with this name")
	ErrRecordChecksumExistsInAnotherRegistry = sdkerrors.Register(ModuleName, 2101, "record with the same checksum exists in another registry")
	ErrDuplicateRecordKey                    = sdkerrors.Register(ModuleName, 2102, "duplicate record key in genesis")
	ErrInvalidGenesisState                   = sdkerrors.Register(ModuleName, 2103, "invalid genesis state")
)
