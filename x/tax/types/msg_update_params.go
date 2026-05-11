package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgUpdateParams) ValidateBasic() error {
	// Validate Authority address
	if msg.Authority == "" {
		return fmt.Errorf("authority address cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	// Validate Tax
	if msg.Tax != "" {
		tax, err := math.LegacyNewDecFromStr(msg.Tax)
		if err != nil {
			return fmt.Errorf("invalid tax: %w", err)
		}
		if tax.IsNegative() {
			return fmt.Errorf("tax cannot be negative")
		}
		if tax.GT(MaxTax) {
			return fmt.Errorf("tax %s cannot exceed maximum of %s", tax, MaxTax)
		}
	}

	// Validate Tax Address
	if msg.TaxAddress != "" {
		_, err = sdk.AccAddressFromBech32(msg.TaxAddress)
		if err != nil {
			return fmt.Errorf("invalid tax address: %w", err)
		}
	}

	return nil
}
