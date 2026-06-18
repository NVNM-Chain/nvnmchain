package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgUpdateParams) ValidateBasic() error {
	// Validate Authority address
	if msg.Authority == "" {
		return fmt.Errorf("Authority address cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return fmt.Errorf("invalid Authority address: %w", err)
	}

	// Validate TaxAddress
	if msg.Admin != "" {
		_, err = sdk.AccAddressFromBech32(msg.Admin)
		if err != nil {
			return fmt.Errorf("invalid admin address: %w", err)
		}
	}

	if msg.AnchoringFee != nil {
		if err := ValidateAnchoringFee(*msg.AnchoringFee); err != nil {
			return err
		}
	}

	return nil
}
