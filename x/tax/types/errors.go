package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/tax module sentinel errors
var (
	ErrInvalidSigner = sdkerrors.Register(ModuleName, 1100, "expected tax address as signer")
)
