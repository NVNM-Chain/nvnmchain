package types_test

import (
	"net/url"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/utilities"
	"github.com/stretchr/testify/require"

	"github.com/NVNM-Chain/nvnmchain/x/anchoring/types"
)

func TestGatewayParsesRegistryNameMatchMode(t *testing.T) {
	for value, want := range map[string]types.RegistryNameMatchMode{
		"REGISTRY_NAME_MATCH_MODE_PREFIX":   types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_PREFIX,
		"REGISTRY_NAME_MATCH_MODE_CONTAINS": types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_CONTAINS,
		"3":                                 types.RegistryNameMatchMode_REGISTRY_NAME_MATCH_MODE_SUFFIX,
	} {
		var req types.QuerySearchRegistriesByNameRequest
		err := runtime.PopulateQueryParameters(&req, url.Values{"name": {"foo"}, "mode": {value}}, utilities.NewDoubleArray(nil))
		require.NoError(t, err, value)
		require.Equal(t, want, req.Mode, value)
	}

	for _, value := range []string{"BOGUS", "99", "-1"} {
		var req types.QuerySearchRegistriesByNameRequest
		err := runtime.PopulateQueryParameters(&req, url.Values{"mode": {value}}, utilities.NewDoubleArray(nil))
		require.Error(t, err, value)
	}
}
