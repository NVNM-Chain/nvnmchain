package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgAddRegistry) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender address cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if err := ValidateRegistryForCreate(msg.Name, msg.Description, msg.Metadata); err != nil {
		return err
	}

	return nil
}
