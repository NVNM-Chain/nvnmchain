package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

const DefaultAdminAddress = "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"

// DefaultAnchoringFeeAmount: 10^16 = 1¢ at 18-decimal USD-pegged denom.
var DefaultAnchoringFeeAmount = math.NewIntWithDecimal(1, 16)

func NewParams(admin string, anchoringFee sdk.Coin) Params {
	return Params{Admin: admin, AnchoringFee: anchoringFee}
}

// ResolveAnchoringFeeDenom fills AnchoringFee.Denom from the registered EVM
// coin denom when the stored value is the InitGenesis/upgrade sentinel.
// Single source of truth for InitGenesis, MsgUpdateParams, and the v1.1.0
// upgrade handler.
func (p *Params) ResolveAnchoringFeeDenom() {
	if p.AnchoringFee.Denom == "" {
		p.AnchoringFee.Denom = evmtypes.GetEVMCoinDenom()
	}
}

// DefaultParams ships an empty AnchoringFee.Denom; anchoring InitGenesis
// and the v1.1.0 upgrade handler resolve it to evmtypes.GetEVMCoinDenom().
func DefaultParams() Params {
	return NewParams(
		DefaultAdminAddress,
		sdk.Coin{Denom: "", Amount: DefaultAnchoringFeeAmount},
	)
}

func (p Params) Validate() error {
	if err := ValidateAdminAddress(p.Admin); err != nil {
		return err
	}
	return ValidateAnchoringFee(p.AnchoringFee)
}

// ValidateAdminAddress validates the admin address.
func ValidateAdminAddress(address string) error {
	if address == "" {
		return fmt.Errorf("admin address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(address); err != nil {
		return fmt.Errorf("invalid admin address: %w", err)
	}
	return nil
}

// ValidateAnchoringFee accepts empty denom (the InitGenesis/upgrade
// sentinel) but requires a positive amount.
func ValidateAnchoringFee(fee sdk.Coin) error {
	if fee.Amount.IsNil() || !fee.Amount.IsPositive() {
		return fmt.Errorf("anchoring fee amount must be positive")
	}
	if fee.Denom == "" {
		return nil
	}
	if err := sdk.ValidateDenom(fee.Denom); err != nil {
		return fmt.Errorf("invalid anchoring fee denom: %w", err)
	}
	return nil
}
