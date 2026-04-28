package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Declare defaults for tax and tax address
var (
	DefaultTax        = "0.4"
	DefaultTaxAddress = "nvnm15m77x4pe6w9vtpuqm22qxu0ds7vn4ehzxt8qca"
	MaxTax            = math.LegacyMustNewDecFromStr("0.4") // 40 %
)

// NewParams creates a new Params instance.
func NewParams(
	taxStr string,
	taxAddress string,
) Params {
	tax := math.LegacyMustNewDecFromStr(taxStr)
	return Params{
		Tax:        tax,
		TaxAddress: taxAddress,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultTax,
		DefaultTaxAddress,
	)
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if err := ValidateTax(p.Tax.String()); err != nil {
		return err
	}
	if err := ValidateTaxAddress(p.TaxAddress); err != nil {
		return err
	}
	return nil
}

// ValidateTax validates the tax.
func ValidateTax(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	tax, err := math.LegacyNewDecFromStr(v)
	if err != nil {
		return fmt.Errorf("invalid tax: %s", err)
	}

	if tax.IsNegative() {
		return fmt.Errorf("tax cannot be negative")
	}

	if tax.GT(MaxTax) {
		return fmt.Errorf("tax cannot exceed maximum of %s", MaxTax)
	}

	return nil
}

// ValidateTaxAddress validates the tax address.
func ValidateTaxAddress(address string) error {
	if address == "" {
		return fmt.Errorf("tax address cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return fmt.Errorf("invalid tax address: %w", err)
	}
	return nil
}
