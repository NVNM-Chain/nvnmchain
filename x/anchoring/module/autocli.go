package anchoring

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	types "github.com/MANTRA-Chain/inveniam/x/anchoring/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "AddRegistry",
					Skip:      true, // use precompile instead
				},
				{
					RpcMethod: "AddRecord",
					Skip:      true, // use precompile instead
				},
				{
					RpcMethod: "UpdateRecordStatus",
					Skip:      true, // use precompile instead
				},
				{
					RpcMethod: "GrantRole",
					Skip:      true, // use precompile instead
				},
				{
					RpcMethod: "RevokeRole",
					Skip:      true, // use precompile instead
				},
				{
					RpcMethod: "UpdateParams",
					Use:       "update-params",
					Skip:      false,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"admin": {
							Name:         "admin",
							Usage:        "admin address for the setting all anchoring",
							DefaultValue: "",
						},
					},
					Short:   "Update the parameters of the anchoring module",
					Example: "inveniamd tx anchoring update-params --admin inveniam1axznhnm82lah8qqvp9hxdad49yx3s5dc2p4pfz",
				},
			},
		},
	}
}
