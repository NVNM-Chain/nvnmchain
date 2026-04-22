package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MaxRegistryNameLen        = 128
	MaxRegistryDescriptionLen = 2048
	MaxRegistryMetadataLen    = 2048
)

// ValidateRegistryForCreate validates registry fields for creation.
func ValidateRegistryForCreate(name, description, metadata string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if err := validateMaxLen("name", name, MaxRegistryNameLen); err != nil {
		return err
	}
	if err := validateMaxLen("description", description, MaxRegistryDescriptionLen); err != nil {
		return err
	}
	if err := validateMaxLen("metadata", metadata, MaxRegistryMetadataLen); err != nil {
		return err
	}
	return nil
}

// ValidateRegistry validates a Registry for genesis.
func ValidateRegistry(r Registry) error {
	if r.Id <= 0 {
		return fmt.Errorf("id cannot be <= 0")
	}
	if r.Creator == "" {
		return fmt.Errorf("creator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(r.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	return ValidateRegistryForCreate(r.Name, r.Description, r.Metadata)
}
