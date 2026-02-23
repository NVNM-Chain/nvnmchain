package types

import "fmt"

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
