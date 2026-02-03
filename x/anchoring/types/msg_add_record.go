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
	if msg.Record.Checksum == "" {
		return fmt.Errorf("checksum cannot be empty")
	}
	if msg.Record.ChecksumAlgo == "" {
		return fmt.Errorf("checksum algorithm cannot be empty")
	}
	if msg.Record.Uri == "" {
		return fmt.Errorf("uri cannot be empty")
	}
	if msg.Record.Metadata == "" || msg.Record.Metadata == "{}" {
		return fmt.Errorf("metadata cannot be empty")
	}
	if msg.Record.Status == "" {
		return fmt.Errorf("status cannot be empty")
	}
	if msg.Record.Registry == "" {
		return fmt.Errorf("registry cannot be empty")
	}
	return nil
}
