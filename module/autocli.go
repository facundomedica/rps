package module

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	rpsv1 "github.com/facundomedica/rps/api/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: rpsv1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Games",
					Use:       "games",
					Short:     "Get all games",
				},
				{
					RpcMethod: "Count",
					Use:       "count",
					Short:     "Get total games count",
				},
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Get the current module parameters",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: rpsv1.Msg_ServiceDesc.ServiceName,
		},
	}
}
