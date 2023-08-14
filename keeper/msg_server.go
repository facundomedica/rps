package keeper

import (
	"context"
	"fmt"
	"strings"

	"github.com/facundomedica/rps"
)

type msgServer struct {
	k Keeper
}

var _ rps.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper Keeper) rps.MsgServer {
	return &msgServer{k: keeper}
}

// CommitMove implements rps.MsgServer.
func (msgServer) CommitMove(context.Context, *rps.MsgCommitMove) (*rps.MsgCommitMoveResponse, error) {
	panic("unimplemented")
}

// NewGame implements rps.MsgServer.
func (msgServer) NewGame(context.Context, *rps.MsgNewGame) (*rps.MsgNewGameResponse, error) {
	panic("unimplemented")
}

// RevealMove implements rps.MsgServer.
func (msgServer) RevealMove(context.Context, *rps.MsgRevealMove) (*rps.MsgRevealMoveResponse, error) {
	panic("unimplemented")
}

// UpdateParams params is defining the handler for the MsgUpdateParams message.
func (ms msgServer) UpdateParams(ctx context.Context, msg *rps.MsgUpdateParams) (*rps.MsgUpdateParamsResponse, error) {
	if _, err := ms.k.addressCodec.StringToBytes(msg.Authority); err != nil {
		return nil, fmt.Errorf("invalid authority address: %w", err)
	}

	if authority := ms.k.GetAuthority(); !strings.EqualFold(msg.Authority, authority) {
		return nil, fmt.Errorf("unauthorized, authority does not match the module's authority: got %s, want %s", msg.Authority, authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	if err := ms.k.Params.Set(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &rps.MsgUpdateParamsResponse{}, nil
}
