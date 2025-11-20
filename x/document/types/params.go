package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Declare defaults for MCA tax and MCA address
const DefaultAdminAddress = "inveniam15m77x4pe6w9vtpuqm22qxu0ds7vn4ehz80mwh8"

// NewParams creates a new Params instance.
func NewParams(
	admin string,
) Params {
	return Params{
		Admin: admin,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(DefaultAdminAddress)
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if err := ValidateAdminAddress(p.Admin); err != nil {
		return err
	}
	return nil
}

// ValidateAdminAddress validates the admin address.
func ValidateAdminAddress(address string) error {
	if address == "" {
		return fmt.Errorf("admin address cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return fmt.Errorf("invalid admin address: %w", err)
	}
	if !strings.HasPrefix(address, "inveniam") {
		return fmt.Errorf("admin address must have 'inveniam' prefix")
	}
	return nil
}
