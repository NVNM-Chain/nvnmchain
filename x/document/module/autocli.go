package document

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	types "github.com/MANTRA-Chain/inveniam/x/document/types"
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
						"admin": {
							Name:         "admin",
							Usage:        "admin address for the setting all documents",
							DefaultValue: "",
						},
					},
					Short:   "Update the parameters of the document module",
					Example: "inveniamd tx document update-params --admin inveniam1axznhnm82lah8qqvp9hxdad49yx3s5dc2p4pfz",
				},
				{
					RpcMethod: "AddDocument",
					Use:       "add-document [path_to_document.json]",
					Short:     "Adds a new document from a JSON file",
					Example:   "inveniamd tx document add-document /path/to/your_document.json",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "document",
						},
					},
				},
			},
		},
	}
}
