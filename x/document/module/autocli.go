package document

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	taxapi "github.com/MANTRA-Chain/inveniam/api/mantrachain/tax/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: taxapi.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              taxapi.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Use:       "update-params",
					Skip:      false,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"admin": {
							Name:         "admin",
							Usage:        "admin address for the setting all documents",
							DefaultValue: "",
						},
					},
					Short:   "Update the parameters of the document module",
					Example: "inveniamd tx document update-params --admin inveniam1axznhnm82lah8qqvp9hxdad49yx3s5dc2p4pfz",
				},
			},
		},
	}
}
