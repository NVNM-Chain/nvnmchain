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

	// Validate McaAddress
	if msg.Admin != "" {
		_, err = sdk.AccAddressFromBech32(msg.Admin)
		if err != nil {
			return fmt.Errorf("invalid admin address: %w", err)
		}
		if !strings.HasPrefix(msg.Admin, "inveniam") {
			return fmt.Errorf("admin address must have 'inveniam' prefix")
		}
	}

	return nil
}
