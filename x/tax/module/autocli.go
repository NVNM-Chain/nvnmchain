package tax

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	types "github.com/MANTRA-Chain/nvnmchain/x/tax/types"
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
					RpcMethod: "UpdateParams",
					Use:       "update-params",
					Skip:      false,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"tax": {
							Name:         "tax",
							Usage:        "tax for the allocation in decimal",
							DefaultValue: "",
						},
						"tax_address": {
							Name:         "tax_address",
							Usage:        "tax address for the allocation",
							DefaultValue: "",
						},
					},
					Short:   "Update the parameters of the tax module",
					Example: "nvnmchaind tx tax update-params --tax 0.4 --tax_address nvnm1axznhnm82lah8qqvp9hxdad49yx3s5dc6h6p3s",
				},
			},
		},
	}
}
