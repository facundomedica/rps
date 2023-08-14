package rps

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	types "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "rps/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgNewGame{}, "rps/MsgNewGame")
	legacy.RegisterAminoMsg(cdc, &MsgCommitMove{}, "rps/MsgCommitMove")
	legacy.RegisterAminoMsg(cdc, &MsgRevealMove{}, "rps/MsgRevealMove")
}

// RegisterInterfaces registers the interfaces types with the interface registry.
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgNewGame{},
		&MsgCommitMove{},
		&MsgRevealMove{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
