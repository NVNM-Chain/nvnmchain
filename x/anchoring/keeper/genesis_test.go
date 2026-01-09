package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"github.com/MANTRA-Chain/inveniam/testutil/keeper"
	"github.com/MANTRA-Chain/inveniam/x/anchoring/types"
	"github.com/stretchr/testify/require"
)

func TestInitGenesisAndExportGenesis(t *testing.T) {
	k, ctx, _ := keeper.AnchoringKeeper(t)

	// Prepare dummy data
	params := types.Params{
		Admin: "cosmos1x0dqq9v6chqeholder",
	}

	registryOne := types.Registry{
		Id:          1,
		Name:        "kyc_registry",
		Description: "KYC document registry",
		Creator:     "cosmos1creator1",
		CreatedAt:   "2024-01-01T00:00:00Z",
	}

	registryTwo := types.Registry{
		Id:          2,
		Name:        "aml_registry",
		Description: "AML document registry",
		Creator:     "cosmos1creator2",
		CreatedAt:   "2024-01-02T00:00:00Z",
	}

	recordOne := types.Record{
		Registry:     "kyc_registry",
		Uri:          "ipfs://QmExample1",
		Checksum:     "sha256hash001",
		ChecksumAlgo: "SHA-256",
		Metadata:     `{"document":"kyc_document_001","figi":"BBG000B9M9V0","individualId":"individual_001"}`,
		Timestamp:    "2024-01-01T10:00:00Z",
		Status:       "active",
		RecordId:     1,
		Index:        1,
		IsLatest:     true,
	}

	recordTwo := types.Record{
		Registry:     "kyc_registry",
		Uri:          "ipfs://QmExample2",
		Checksum:     "sha256hash002",
		ChecksumAlgo: "SHA-256",
		Metadata:     `{"document":"kyc_document_002","figi":"BBG000B9M9V1","individualId":"individual_002"}`,
		Timestamp:    "2024-01-01T11:00:00Z",
		Status:       "active",
		RecordId:     2,
		Index:        1,
		IsLatest:     true,
	}

	recordThree := types.Record{
		Registry:     "aml_registry",
		Uri:          "ipfs://QmExample3",
		Checksum:     "sha256hash003",
		ChecksumAlgo: "SHA-256",
		Metadata:     `{"document":"aml_document_001","figi":"BBG000B9M9V2","individualId":"individual_003"}`,
		Timestamp:    "2024-01-02T10:00:00Z",
		Status:       "active",
		RecordId:     1,
		Index:        1,
		IsLatest:     true,
	}

	// Prepare encoded role keys
	// Registry-level admin role for creator1
	registryAdminKey1 := collections.Join3(uint64(1), "cosmos1creator1", "")
	registryAdminKeyBytes1 := encodeTripleKey(t, registryAdminKey1)

	// Registry-level admin role for creator2
	registryAdminKey2 := collections.Join3(uint64(2), "cosmos1creator2", "")
	registryAdminKeyBytes2 := encodeTripleKey(t, registryAdminKey2)

	// Record-level editor role for user1 on sha256hash001
	docEditorKey := collections.Join3(uint64(1), "cosmos1user1", "sha256hash001")
	docEditorKeyBytes := encodeTripleKey(t, docEditorKey)

	// Registry-level editor role for user2 on kyc_registry
	registryEditorKey := collections.Join3(uint64(1), "cosmos1user2", "")
	registryEditorKeyBytes := encodeTripleKey(t, registryEditorKey)

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
	}

	// Test InitGenesis
	err := k.InitGenesis(ctx, genState)
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
	require.Equal(t, "sha256hash001", record1.Checksum)

	record2, err := k.Records.Get(ctx, collections.Join3(uint64(1), uint64(2), uint64(1)))
	require.NoError(t, err)
	require.Equal(t, "sha256hash002", record2.Checksum)

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
	docEditorRole, err := k.Roles.Get(ctx, collections.Join3(uint64(1), "cosmos1user1", "sha256hash001"))
	require.NoError(t, err)
	require.Equal(t, "editor", docEditorRole)

	registryEditorRole, err := k.Roles.Get(ctx, collections.Join3(uint64(1), "cosmos1user2", ""))
	require.NoError(t, err)
	require.Equal(t, "editor", registryEditorRole)

	// Test ExportGenesis
	exportedState, err := k.ExportGenesis(ctx)
	require.NoError(t, err)

	// Verify exported data
	require.Equal(t, params.Admin, exportedState.Params.Admin)
	require.Len(t, exportedState.Registries, 2)
	require.Len(t, exportedState.Records, 3)

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
}

func TestInitGenesisEmpty(t *testing.T) {
	k, ctx, _ := keeper.AnchoringKeeper(t)

	// Test with empty genesis state
	emptyState := types.GenesisState{
		Params: types.Params{
			Admin: "cosmos1x0dqq9v6chqeholder",
		},
		Registries: map[uint64]types.Registry{},
		Records:    []types.Record{},
		Roles:      map[string]string{},
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
