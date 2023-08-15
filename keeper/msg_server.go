package keeper

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/collections"
	"github.com/facundomedica/rps"
	"github.com/facundomedica/rps/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	k Keeper
}

var _ rps.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper Keeper) rps.MsgServer {
	return &msgServer{k: keeper}
}

// NewGame implements rps.MsgServer.
func (ms msgServer) NewGame(ctx context.Context, msg *rps.MsgNewGame) (*rps.MsgNewGameResponse, error) {
	if msg.EntryFee.IsZero() {
		return nil, fmt.Errorf("entry fee must be positive")
	}

	playerAddr, err := ms.k.addressCodec.StringToBytes(msg.Player)
	if err != nil {
		return nil, fmt.Errorf("invalid player address: %w", err)
	}

	err = ms.k.bankKeeper.SendCoinsFromAccountToModule(ctx, playerAddr, rps.ModuleName, sdk.NewCoins(msg.EntryFee))
	if err != nil {
		return nil, err
	}

	// create game
	gid, err := ms.k.GameID.Next(ctx)
	if err != nil {
		return nil, err
	}

	params, err := ms.k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	game := rps.Game{
		Id:            gid,
		EntryFee:      msg.EntryFee,
		CommitTimeout: sdkCtx.BlockTime().Add(time.Second * time.Duration(params.CommitTimeout)),
	}

	err = ms.k.Games.Set(ctx, gid, game)
	if err != nil {
		return nil, err
	}

	// store move commit
	commit := rps.MoveCommit{
		Commit:    msg.Commit,
		CreatedAt: sdkCtx.BlockTime(),
	}

	err = ms.k.MoveCommits.Set(ctx, collections.Join(gid, playerAddr), commit)
	if err != nil {
		return nil, err
	}

	return &rps.MsgNewGameResponse{GameId: gid}, nil
}

// CommitMove implements rps.MsgServer.
func (ms msgServer) CommitMove(ctx context.Context, msg *rps.MsgCommitMove) (*rps.MsgCommitMoveResponse, error) {
	playerAddr, err := ms.k.addressCodec.StringToBytes(msg.Player)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	// check if the game exists, check if the commit timeout hasn't passed and if the player is already in the game
	game, err := ms.k.Games.Get(ctx, msg.GameId)
	if err != nil {
		return nil, err
	}

	if !game.CommitTimeout.IsZero() && sdk.UnwrapSDKContext(ctx).BlockTime().After(game.CommitTimeout) {
		return nil, errors.New("commit timeout has passed")
	}

	alreadyInGame, err := ms.k.MoveCommits.Has(ctx, collections.Join(msg.GameId, playerAddr))
	if err != nil {
		return nil, err
	}

	if alreadyInGame {
		return nil, errors.New("player already in game")
	}

	err = ms.k.bankKeeper.SendCoinsFromAccountToModule(ctx, playerAddr, rps.ModuleName, sdk.NewCoins(game.EntryFee))
	if err != nil {
		return nil, err
	}

	// check if the game is full
	players := 0
	rng := collections.NewPrefixedPairRange[uint64, []byte](msg.GameId)
	err = ms.k.MoveCommits.Walk(ctx, rng, func(key collections.Pair[uint64, []byte], value rps.MoveCommit) (stop bool, err error) {
		players++
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	if players == 2 {
		return nil, errors.New("game is full, sorry")
	}

	// now that we've checked everything, we can store the move commitment
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	commit := rps.MoveCommit{
		Commit:    msg.Commit,
		CreatedAt: sdkCtx.BlockTime(),
	}

	if err := ms.k.MoveCommits.Set(ctx, collections.Join(msg.GameId, playerAddr), commit); err != nil {
		return nil, err
	}

	return &rps.MsgCommitMoveResponse{}, nil
}

// RevealMove implements rps.MsgServer.
func (ms msgServer) RevealMove(ctx context.Context, msg *rps.MsgRevealMove) (*rps.MsgRevealMoveResponse, error) {
	playerAddr, err := ms.k.addressCodec.StringToBytes(msg.Player)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	if !utils.MoveIsValid(msg.Move) {
		return nil, errors.New("invalid move")
	}

	// check if the game exists and that the reveal timeout hasn't passed
	game, err := ms.k.Games.Get(ctx, msg.GameId)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !game.RevealTimeout.IsZero() && sdkCtx.BlockTime().After(game.RevealTimeout) {
		return nil, errors.New("reveal timeout has passed")
	}

	// check if the player is part of the game
	moveCommit, err := ms.k.MoveCommits.Get(ctx, collections.Join(msg.GameId, playerAddr))
	if err != nil {
		return nil, err
	}

	// check if the game is full, if it is, we allow the player to reveal their move
	players := 0
	rng := collections.NewPrefixedPairRange[uint64, []byte](msg.GameId)
	err = ms.k.MoveCommits.Walk(ctx, rng, func(key collections.Pair[uint64, []byte], value rps.MoveCommit) (stop bool, err error) {
		players++
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	if players != 2 {
		return nil, errors.New("please wait until the game is full")
	}

	params, err := ms.k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	// store reveal timeout if it's the first reveal
	if game.RevealTimeout.IsZero() {
		game.RevealTimeout = sdkCtx.BlockTime().Add(time.Second * time.Duration(params.RevealTimeout))
		if err := ms.k.Games.Set(ctx, msg.GameId, game); err != nil {
			return nil, err
		}
	}

	// check if the move has already been revealed
	revealed, err := ms.k.MoveReveals.Has(ctx, collections.Join(msg.GameId, playerAddr))
	if err != nil {
		return nil, err
	}

	if revealed {
		return nil, errors.New("move already revealed")
	}

	// all good, let's reveal the move
	// calculate the move's commitment, must match the one stored
	commit := utils.CalculateCommitment(msg.Move, msg.Salt)
	if commit != moveCommit.Commit {
		return nil, errors.New("move doesn't match commitment, are you a cheater?")
	}

	// store the reveal
	reveal := rps.MoveReveal{
		Move:      msg.Move,
		Salt:      msg.Salt,
		CreatedAt: sdkCtx.BlockTime(),
	}

	err = ms.k.MoveReveals.Set(ctx, collections.Join(msg.GameId, playerAddr), reveal)
	if err != nil {
		return nil, err
	}

	return &rps.MsgRevealMoveResponse{}, nil
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
