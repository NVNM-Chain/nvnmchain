package types

import (
	"fmt"
	"strings"

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
		if !strings.HasPrefix(msg.Admin, "nvnm") {
			return fmt.Errorf("admin address must have 'nvnm' prefix")
		}
	}

	return nil
}
