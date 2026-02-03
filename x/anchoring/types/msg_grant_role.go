package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgGrantRole) ValidateBasic() error {
	if msg.Admin == "" {
		return fmt.Errorf("admin address cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		return fmt.Errorf("invalid admin address: %w", err)
	}
	if msg.Address == "" {
		return fmt.Errorf("grantee address cannot be empty")
	}
	_, err = sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return fmt.Errorf("invalid grantee address: %w", err)
	}
	if msg.Role == "" {
		return fmt.Errorf("role cannot be empty")
	}

	return nil
}
