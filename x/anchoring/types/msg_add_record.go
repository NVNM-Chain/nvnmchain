package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgAddRecord) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender address cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.Record == nil {
		return fmt.Errorf("record cannot be nil")
	}
	if err := ValidateRecordForCreate(*msg.Record); err != nil {
		return err
	}
	return nil
}
