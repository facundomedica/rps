package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	storetypes "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/facundomedica/rps"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	addressCodec address.Codec

	// authority is the address capable of executing a MsgUpdateParams and other authority-gated message.
	// typically, this should be the x/gov module account.
	authority string

	// state management
	Schema      collections.Schema
	Params      collections.Item[rps.Params]
	GameID      collections.Sequence
	Games       collections.Map[uint64, rps.Game]
	MoveCommits collections.Map[collections.Pair[uint64, []byte], rps.MoveCommit]
	MoveReveals collections.Map[collections.Pair[uint64, []byte], rps.MoveReveal]
}

// NewKeeper creates a new Keeper instance
func NewKeeper(cdc codec.BinaryCodec, addressCodec address.Codec, storeService storetypes.KVStoreService, authority string) Keeper {
	if _, err := addressCodec.StringToBytes(authority); err != nil {
		panic(fmt.Errorf("invalid authority address: %w", err))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,
		Params:       collections.NewItem(sb, rps.ParamsKey, "params", codec.CollValue[rps.Params](cdc)),
		GameID:       collections.NewSequence(sb, rps.GameIDKey, "game_id"),
		Games:        collections.NewMap(sb, rps.GamesKey, "games", collections.Uint64Key, codec.CollValue[rps.Game](cdc)),
		MoveCommits:  collections.NewMap(sb, rps.MoveCommitKey, "move_commits", collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey), codec.CollValue[rps.MoveCommit](cdc)),
		MoveReveals:  collections.NewMap(sb, rps.MoveRevealKey, "move_reveals", collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey), codec.CollValue[rps.MoveReveal](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) GenesisHandler() appmodule.HasGenesis {
	return k.Schema
}
