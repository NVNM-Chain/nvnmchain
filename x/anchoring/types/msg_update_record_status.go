package types

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgUpdateRecordStatus) ValidateBasic() error {
	if msg.Editor == "" {
		return fmt.Errorf("Editor address cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(msg.Editor)
	if err != nil {
		return fmt.Errorf("invalid Editor address: %w", err)
	}
	if msg.RecordId == 0 {
		return fmt.Errorf("record ID cannot be zero")
	}
	if err := ValidateRecordStatus(msg.Status); err != nil {
		return err
	}
	if msg.RegistryId == 0 {
		return errors.New(ErrMsgRegistryIDZero)
	}
	if msg.Index == 0 {
		return fmt.Errorf("index cannot be zero")
	}
	return nil
}
