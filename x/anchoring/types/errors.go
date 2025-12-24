package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/anchoring module sentinel errors
var (
	ErrRegistryExists                        = sdkerrors.Register(ModuleName, 2100, "registry already exists with this name")
	ErrRecordChecksumExistsInAnotherRegistry = sdkerrors.Register(ModuleName, 2101, "record with the same checksum exists in another registry")
)
