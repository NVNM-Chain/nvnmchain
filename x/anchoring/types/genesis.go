package types

import "fmt"

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	names := make(map[string]struct{}, len(gs.Registries))
	for mapKey, registry := range gs.Registries {
		if mapKey != registry.Id {
			return fmt.Errorf("registry map key %d does not match registry id %d", mapKey, registry.Id)
		}
		if err := ValidateRegistry(registry); err != nil {
			return fmt.Errorf("invalid registry %d: %w", registry.Id, err)
		}
		if _, dup := names[registry.Name]; dup {
			return fmt.Errorf("duplicate registry name: %s", registry.Name)
		}
		names[registry.Name] = struct{}{}
	}

	type recordKey struct {
		registry string
		recordID uint64
		index    uint64
	}
	type versionGroupKey struct {
		registry string
		recordID uint64
	}
	seenRecords := make(map[recordKey]struct{}, len(gs.Records))
	groups := make(map[versionGroupKey]*RecordVersionGroup)
	for i, record := range gs.Records {
		if err := ValidateRecord(record); err != nil {
			return fmt.Errorf("invalid record at index %d: %w", i, err)
		}
		if _, exists := names[record.Registry]; !exists {
			return fmt.Errorf("record at index %d references unknown registry %q", i, record.Registry)
		}
		rk := recordKey{record.Registry, record.RecordId, record.Index}
		if _, dup := seenRecords[rk]; dup {
			return fmt.Errorf("duplicate record key (registry=%s, record_id=%d, index=%d)", record.Registry, record.RecordId, record.Index)
		}
		seenRecords[rk] = struct{}{}

		gk := versionGroupKey{record.Registry, record.RecordId}
		if groups[gk] == nil {
			groups[gk] = &RecordVersionGroup{}
		}
		groups[gk].Add(record)
	}
	for gk, g := range groups {
		label := fmt.Sprintf("record group (registry=%s, record_id=%d)", gk.registry, gk.recordID)
		if err := g.ValidateIsLatest(label); err != nil {
			return err
		}
	}

	for key, adminRole := range gs.RoleAdmins {
		if key == "" {
			return fmt.Errorf("role_admins: key cannot be empty")
		}
		if len(adminRole) == 0 {
			return fmt.Errorf("role_admins: admin role bytes cannot be empty (key=%q)", key)
		}
	}

	for key := range gs.RoleMembers {
		if key == "" {
			return fmt.Errorf("role_members: key cannot be empty")
		}
	}

	return nil
}
