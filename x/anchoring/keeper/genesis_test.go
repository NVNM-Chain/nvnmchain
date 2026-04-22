package keeper_test

import (
	"strings"
	"testing"

	"cosmossdk.io/collections"
	"github.com/MANTRA-Chain/nvnmchain/testutil/keeper"
	"github.com/MANTRA-Chain/nvnmchain/testutil/sample"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/rbac"
	"github.com/MANTRA-Chain/nvnmchain/x/anchoring/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

const (
	testChecksum1   = "0000000000000000000000000000000000000000000000000000000000000001"
	testChecksum2   = "0000000000000000000000000000000000000000000000000000000000000002"
	testChecksum3   = "0000000000000000000000000000000000000000000000000000000000000003"
	testChecksum999 = "0000000000000000000000000000000000000000000000000000000000000999"
)

func TestInitGenesisAndExportGenesis(t *testing.T) {
	k, ctx, _ := keeper.AnchoringKeeper(t)

	// Prepare dummy data
	params := types.Params{
		Admin: "cosmos1x0dqq9v6chqeholder",
	}

	registryOne := testRegistry(1, "kyc_registry", "KYC document registry", "cosmos1creator1", "2024-01-01T00:00:00Z")
	registryTwo := testRegistry(2, "aml_registry", "AML document registry", "cosmos1creator2", "2024-01-02T00:00:00Z")

	recordOne := testRecord("kyc_registry", "ipfs://QmExample1", testChecksum1, `{"document":"kyc_document_001","figi":"BBG000B9M9V0","individualId":"individual_001"}`, "2024-01-01T10:00:00Z", 1)
	recordTwo := testRecord("kyc_registry", "ipfs://QmExample2", testChecksum2, `{"document":"kyc_document_002","figi":"BBG000B9M9V1","individualId":"individual_002"}`, "2024-01-01T11:00:00Z", 2)
	recordThree := testRecord("aml_registry", "ipfs://QmExample3", testChecksum3, `{"document":"aml_document_001","figi":"BBG000B9M9V2","individualId":"individual_003"}`, "2024-01-02T10:00:00Z", 1)

	// Prepare encoded role keys
	// Registry-level admin role for creator1
	registryAdminKey1 := collections.Join3(uint64(1), "cosmos1creator1", "")
	registryAdminKeyBytes1 := encodeTripleKey(t, registryAdminKey1)

	// Registry-level admin role for creator2
	registryAdminKey2 := collections.Join3(uint64(2), "cosmos1creator2", "")
	registryAdminKeyBytes2 := encodeTripleKey(t, registryAdminKey2)

	// Record-level editor role for user1 on sha256hash001
	docEditorKey := collections.Join3(uint64(1), "cosmos1user1", testChecksum1)
	docEditorKeyBytes := encodeTripleKey(t, docEditorKey)

	// Registry-level editor role for user2 on kyc_registry
	registryEditorKey := collections.Join3(uint64(1), "cosmos1user2", "")
	registryEditorKeyBytes := encodeTripleKey(t, registryEditorKey)

	adminRole := k.RegistryRole(1, "admin")
	adminRoleKeyBytes := encodeRBACRoleAdminKey(t, adminRole)

	memberAddr1, err := sdk.AccAddressFromBech32(sample.AccAddress())
	require.NoError(t, err)
	rbacMemberKey1 := collections.Join(adminRole, memberAddr1)
	rbacMemberKeyBytes1 := encodeRBACRoleMemberKey(t, rbacMemberKey1)

	memberAddr2, err := sdk.AccAddressFromBech32(sample.AccAddress())
	require.NoError(t, err)
	rbacMemberKey2 := collections.Join(adminRole, memberAddr2)
	rbacMemberKeyBytes2 := encodeRBACRoleMemberKey(t, rbacMemberKey2)

	genState := types.GenesisState{
		Params: params,
		Registries: map[uint64]types.Registry{
			1: registryOne,
			2: registryTwo,
		},
		Records: []types.Record{recordOne, recordTwo, recordThree},
		Roles: map[string]string{
			registryAdminKeyBytes1: "admin",
			registryAdminKeyBytes2: "admin",
			docEditorKeyBytes:      "editor",
			registryEditorKeyBytes: "editor",
		},
		RoleAdmins: map[string][]byte{
			adminRoleKeyBytes: adminRole.Bytes(),
		},
		RoleMembers: map[string][]byte{
			rbacMemberKeyBytes1: {},
			rbacMemberKeyBytes2: {},
		},
	}

	// Test InitGenesis
	err = k.InitGenesis(ctx, genState)
	require.NoError(t, err)

	// Verify params were set
	storedParams, err := k.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, params.Admin, storedParams.Admin)

	// Verify registries were stored
	storedRegistry1, err := k.Registries.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, registryOne.Name, storedRegistry1.Name)
	require.Equal(t, registryOne.Description, storedRegistry1.Description)
	require.Equal(t, registryOne.Creator, storedRegistry1.Creator)

	storedRegistry2, err := k.Registries.Get(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, registryTwo.Name, storedRegistry2.Name)

	// Verify registry names mapping
	regId1, err := k.RegistryIdByName.Get(ctx, "kyc_registry")
	require.NoError(t, err)
	require.Equal(t, uint64(1), regId1)

	regId2, err := k.RegistryIdByName.Get(ctx, "aml_registry")
	require.NoError(t, err)
	require.Equal(t, uint64(2), regId2)

	// Verify registry count
	regCount, err := k.RegistryCount.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), regCount)

	// Verify records were stored
	storedRecord1, err := k.Records.Get(ctx, collections.Join3(uint64(1), recordOne.RecordId, uint64(1)))
	require.NoError(t, err)
	require.Equal(t, recordOne.Metadata, storedRecord1.Metadata)
	require.Equal(t, recordOne.Status, storedRecord1.Status)

	storedRecord2, err := k.Records.Get(ctx, collections.Join3(uint64(1), recordTwo.RecordId, uint64(1)))
	require.NoError(t, err)
	require.Equal(t, recordTwo.Metadata, storedRecord2.Metadata)

	// Verify record indices
	recordIndex1, err := k.RecordIndices.Get(ctx, collections.Join(uint64(1), uint64(1)))
	require.NoError(t, err)
	require.Equal(t, uint64(1), recordIndex1)

	// Verify records by registry mapping
	record1, err := k.Records.Get(ctx, collections.Join3(uint64(1), uint64(1), uint64(1)))
	require.NoError(t, err)
	require.Equal(t, testChecksum1, record1.Checksum)

	record2, err := k.Records.Get(ctx, collections.Join3(uint64(1), uint64(2), uint64(1)))
	require.NoError(t, err)
	require.Equal(t, testChecksum2, record2.Checksum)

	// Verify record counts by registry
	count1, err := k.RecordsCountByRegistry.Get(ctx, uint64(1))
	require.NoError(t, err)
	require.Equal(t, uint64(2), count1)

	count2, err := k.RecordsCountByRegistry.Get(ctx, uint64(2))
	require.NoError(t, err)
	require.Equal(t, uint64(1), count2)

	// Verify creator roles
	creatorRole1, err := k.Roles.Get(ctx, collections.Join3(uint64(1), "cosmos1creator1", ""))
	require.NoError(t, err)
	require.Equal(t, "admin", creatorRole1)

	creatorRole2, err := k.Roles.Get(ctx, collections.Join3(uint64(2), "cosmos1creator2", ""))
	require.NoError(t, err)
	require.Equal(t, "admin", creatorRole2)

	// Verify imported roles
	docEditorRole, err := k.Roles.Get(ctx, collections.Join3(uint64(1), "cosmos1user1", testChecksum1))
	require.NoError(t, err)
	require.Equal(t, "editor", docEditorRole)

	registryEditorRole, err := k.Roles.Get(ctx, collections.Join3(uint64(1), "cosmos1user2", ""))
	require.NoError(t, err)
	require.Equal(t, "editor", registryEditorRole)

	storedAdminRole, err := k.RBAC.RoleAdmins.Get(ctx, adminRole)
	require.NoError(t, err)
	require.Equal(t, adminRole.Bytes(), storedAdminRole)

	hasMember1, err := k.RBAC.RoleMembers.Has(ctx, rbacMemberKey1)
	require.NoError(t, err)
	require.True(t, hasMember1)

	hasMember2, err := k.RBAC.RoleMembers.Has(ctx, rbacMemberKey2)
	require.NoError(t, err)
	require.True(t, hasMember2)

	// Test ExportGenesis
	exportedState, err := k.ExportGenesis(ctx)
	require.NoError(t, err)

	// Verify exported data
	require.Equal(t, params.Admin, exportedState.Params.Admin)
	require.Len(t, exportedState.Registries, 2)
	require.Len(t, exportedState.Records, 3)
	require.Len(t, exportedState.RoleAdmins, 1)
	require.Len(t, exportedState.RoleMembers, 2)

	// Verify exported registries
	require.Equal(t, registryOne.Name, exportedState.Registries[1].Name)
	require.Equal(t, registryTwo.Name, exportedState.Registries[2].Name)

	// Verify exported records contain the expected metadata
	// Note: ExportGenesis uses encoded keys, so we verify by checking all records
	recordMetadata := make(map[string]bool)
	for _, record := range exportedState.Records {
		recordMetadata[record.Metadata] = true
	}
	require.True(t, recordMetadata[recordOne.Metadata])
	require.True(t, recordMetadata[recordTwo.Metadata])
	require.True(t, recordMetadata[recordThree.Metadata])

	require.Equal(t, adminRole.Bytes(), exportedState.RoleAdmins[adminRoleKeyBytes])

	_, ok := exportedState.RoleMembers[rbacMemberKeyBytes1]
	require.True(t, ok)
	_, ok = exportedState.RoleMembers[rbacMemberKeyBytes2]
	require.True(t, ok)
}

func TestInitGenesisEmpty(t *testing.T) {
	k, ctx, _ := keeper.AnchoringKeeper(t)

	// Test with empty genesis state
	emptyState := types.GenesisState{
		Params: types.Params{
			Admin: "cosmos1x0dqq9v6chqeholder",
		},
		Registries:  map[uint64]types.Registry{},
		Records:     []types.Record{},
		Roles:       map[string]string{},
		RoleAdmins:  map[string][]byte{},
		RoleMembers: map[string][]byte{},
	}

	err := k.InitGenesis(ctx, emptyState)
	require.NoError(t, err)

	// Verify params were set
	storedParams, err := k.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, emptyState.Params.Admin, storedParams.Admin)

	// Verify registry count is 0
	regCount, err := k.RegistryCount.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(0), regCount)

	// Export and verify empty state
	exportedState, err := k.ExportGenesis(ctx)
	require.NoError(t, err)
	require.Empty(t, exportedState.Registries)
	require.Empty(t, exportedState.Records)
}

func TestInitGenesisRejectsDuplicateRecordKeys(t *testing.T) {
	k, ctx, _ := keeper.AnchoringKeeper(t)
	registry := testRegistry(1, "kyc_registry", "KYC document registry", "cosmos1creator1", "2024-01-01T00:00:00Z")
	recordOne := testRecord("kyc_registry", "ipfs://QmExample1", testChecksum1, `{"document":"kyc_document_001"}`, "2024-01-01T10:00:00Z", 1)
	recordDuplicateKey := testRecord("kyc_registry", "ipfs://QmExample2", testChecksum999, `{"document":"kyc_document_999"}`, "2024-01-01T11:00:00Z", 1)

	genState := types.GenesisState{
		Params: types.Params{
			Admin: "cosmos1x0dqq9v6chqeholder",
		},
		Registries: map[uint64]types.Registry{
			1: registry,
		},
		Records: []types.Record{recordOne, recordDuplicateKey},
		Roles:   map[string]string{},
	}

	err := k.InitGenesis(ctx, genState)
	require.ErrorIs(t, err, types.ErrDuplicateRecordKey)
	require.Contains(t, err.Error(), "registry_id=1")
	require.Contains(t, err.Error(), "record_id=1")
	require.Contains(t, err.Error(), "index=1")
}

func TestInitGenesisRejectsOversizedRecordFields(t *testing.T) {
	k, ctx, _ := keeper.AnchoringKeeper(t)

	registry := testRegistry(1, "kyc_registry", "KYC document registry", "cosmos1creator1", "2024-01-01T00:00:00Z")
	record := testRecord(
		"kyc_registry",
		"ipfs://QmExample1",
		testChecksum1,
		strings.Repeat("a", types.MaxRecordMetadataLen+1),
		"2024-01-01T10:00:00Z",
		1,
	)

	genState := types.GenesisState{
		Params: types.Params{
			Admin: "cosmos1x0dqq9v6chqeholder",
		},
		Registries: map[uint64]types.Registry{
			1: registry,
		},
		Records: []types.Record{record},
		Roles:   map[string]string{},
	}

	err := k.InitGenesis(ctx, genState)
	require.Error(t, err)
	require.Contains(t, err.Error(), "metadata exceeds max length")
}

func testRegistry(id uint64, name, description, creator, createdAt string) types.Registry {
	return types.Registry{
		Id:          id,
		Name:        name,
		Description: description,
		Creator:     creator,
		CreatedAt:   createdAt,
	}
}

func testRecord(registry, uri, checksum, metadata, timestamp string, recordID uint64) types.Record {
	return types.Record{
		Registry:     registry,
		Uri:          uri,
		Checksum:     checksum,
		ChecksumAlgo: "SHA-256",
		Metadata:     metadata,
		Timestamp:    timestamp,
		Status:       "active",
		RecordId:     recordID,
		Index:        1,
		IsLatest:     true,
	}
}

// encodeTripleKey encodes a triple key for testing genesis roles
func encodeTripleKey(t *testing.T, key collections.Triple[uint64, string, string]) string {
	t.Helper()
	k, _, _ := keeper.AnchoringKeeper(t)
	size := k.Roles.KeyCodec().Size(key)
	buffer := make([]byte, size)
	n, err := k.Roles.KeyCodec().Encode(buffer, key)
	require.NoError(t, err)
	return string(buffer[:n])
}

func encodeRBACRoleAdminKey(t *testing.T, key rbac.Role) string {
	t.Helper()
	k, _, _ := keeper.AnchoringKeeper(t)
	size := k.RBAC.RoleAdmins.KeyCodec().Size(key)
	buffer := make([]byte, size)
	n, err := k.RBAC.RoleAdmins.KeyCodec().Encode(buffer, key)
	require.NoError(t, err)
	return string(buffer[:n])
}

func encodeRBACRoleMemberKey(t *testing.T, key collections.Pair[rbac.Role, sdk.AccAddress]) string {
	t.Helper()
	k, _, _ := keeper.AnchoringKeeper(t)
	size := k.RBAC.RoleMembers.KeyCodec().Size(key)
	buffer := make([]byte, size)
	n, err := k.RBAC.RoleMembers.KeyCodec().Encode(buffer, key)
	require.NoError(t, err)
	return string(buffer[:n])
}
