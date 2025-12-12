package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgRemoveDocument) ValidateBasic() error {
	// Validate Authority address
	if msg.Sender == "" {
		return fmt.Errorf("Sender address cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return fmt.Errorf("invalid Sender address: %w", err)
	}

	if msg.Denom == "" {
		return fmt.Errorf("Denom cannot be empty")
	}

	// Validate index address
	if msg.Index <= 0 {
		return fmt.Errorf("Index must be greater than zero")
	}

	return nil
}
