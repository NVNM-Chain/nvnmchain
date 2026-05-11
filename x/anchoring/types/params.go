package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Declare defaults for tax and tax address
const DefaultAdminAddress = "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"

// DefaultAnchoringFeeDenom is a placeholder. The IBC mantraUSD hash is
// runtime-dependent — production chains MUST override AnchoringFee in
// genesis or via MsgUpdateParams; the cap silently no-ops when the denom
// doesn't match what users pay.
const DefaultAnchoringFeeDenom = "anvnm"

// DefaultAnchoringFeeAmount is 10^16 — 1¢ at 18-decimal USD-pegged denom.
var DefaultAnchoringFeeAmount = math.NewIntWithDecimal(1, 16)

// NewParams creates a new Params instance.
func NewParams(
	admin string,
	anchoringFee sdk.Coin,
) Params {
	return Params{
		Admin:        admin,
		AnchoringFee: anchoringFee,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultAdminAddress,
		sdk.NewCoin(DefaultAnchoringFeeDenom, DefaultAnchoringFeeAmount),
	)
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if err := ValidateAdminAddress(p.Admin); err != nil {
		return err
	}
	if err := ValidateAnchoringFee(p.AnchoringFee); err != nil {
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
	return nil
}

// ValidateAnchoringFee enforces a positive, non-zero fee Coin with a valid denom.
func ValidateAnchoringFee(fee sdk.Coin) error {
	if err := fee.Validate(); err != nil {
		return fmt.Errorf("invalid anchoring fee: %w", err)
	}
	if fee.Amount.IsZero() {
		return fmt.Errorf("anchoring fee amount must be positive")
	}
	return nil
}
