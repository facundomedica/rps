package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	storetypes "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/facundomedica/rps"
	expectedkeepers "github.com/facundomedica/rps/expected_keepers"
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

	// other keepers
	bankKeeper expectedkeepers.BankKeeper
}

// NewKeeper creates a new Keeper instance
func NewKeeper(cdc codec.BinaryCodec, addressCodec address.Codec, storeService storetypes.KVStoreService, bk expectedkeepers.BankKeeper, authority string) Keeper {
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
	k.bankKeeper = bk

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) GenesisHandler() appmodule.HasGenesis {
	return k.Schema
}

func (k Keeper) EndBlocker(ctx context.Context) error {
	/* go through all games and:
	- if the commit timeout has passed, delete the game and refund the entry fee
	- if both reveals are available, delete the game and pay the winner
	- if the reveal timeout has passed, delete the game and pay the only player that revealed
	*/

	gamesToDelete := []uint64{}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	fmt.Println("ACACACA")
	err := k.Games.Walk(ctx, nil, func(id uint64, game rps.Game) (bool, error) {
		now := sdkCtx.BlockTime()

		// players that committed
		playersCommited := [][]byte{}
		commits := []rps.MoveCommit{}
		err := k.MoveCommits.Walk(
			ctx,
			collections.NewPrefixedPairRange[uint64, []byte](id),
			func(key collections.Pair[uint64, []byte], commit rps.MoveCommit) (bool, error) {
				playersCommited = append(playersCommited, key.K2())
				commits = append(commits, commit)
				return false, nil
			},
		)
		if err != nil {
			return false, err
		}
		fmt.Println("ACACACA 22222")
		// if the game has less than 2 players and the commit timeout has passed, delete the game and refund the entry fee
		if len(playersCommited) < 2 && now.After(game.CommitTimeout) {
			gamesToDelete = append(gamesToDelete, id)
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, rps.ModuleName, playersCommited[0], sdk.NewCoins(game.EntryFee)); err != nil {
				return false, err
			}
			return false, nil
		}

		// no player revealed so there's no reveal timeout set yet
		if game.RevealTimeout.IsZero() {
			return false, nil
		}

		fmt.Println("ACACACA 33333")
		// now let's check for reveals
		playersRevealed := [][]byte{}
		reveals := []rps.MoveReveal{}
		err = k.MoveReveals.Walk(
			ctx,
			collections.NewPrefixedPairRange[uint64, []byte](id),
			func(key collections.Pair[uint64, []byte], reveal rps.MoveReveal) (bool, error) {
				playersRevealed = append(playersRevealed, key.K2())
				reveals = append(reveals, reveal)
				return false, nil
			},
		)
		if err != nil {
			return false, err
		}

		fmt.Println("ACACACA 4444")

		// revealTimedOut := now.After(game.RevealTimeout)
		// if the reveal timeout hasn't passed and less than 2 players revealed, let's wait
		if len(playersRevealed) < 2 && now.Before(game.RevealTimeout) {
			return false, nil
		}

		// this game is over, let's delete it
		gamesToDelete = append(gamesToDelete, id)

		// given that 2 players committed, the prize is the entry fee times 2
		prize := sdk.NewCoins(sdk.NewCoin(game.EntryFee.Denom, game.EntryFee.Amount.MulRaw(2)))

		fmt.Println("ACACACA 5555")

		// now either 2 players revealed or the reveal timeout has passed
		// if a single player revealed, they win by default
		if len(playersRevealed) == 1 {
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, rps.ModuleName, playersRevealed[0], prize); err != nil {
				return false, err
			}
			return false, nil
		}

		// if both players revealed, let's decide the winner (or winners in case of a draw)
		winners := decideWinner(playersRevealed[0], playersRevealed[1], reveals[0].Move, reveals[1].Move)
		if len(winners) == 1 {
			// a single winner takes all
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, rps.ModuleName, winners[0], prize); err != nil {
				return false, err
			}
		} else if len(winners) == 2 {
			// draw, refund both players
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, rps.ModuleName, winners[0], sdk.NewCoins(game.EntryFee)); err != nil {
				return false, err
			}

			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, rps.ModuleName, winners[1], sdk.NewCoins(game.EntryFee)); err != nil {
				return false, err
			}
		}

		return false, nil
	})
	if err != nil {
		return err
	}

	// delete completed games
	for _, id := range gamesToDelete {
		if err := k.Games.Remove(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

func decideWinner(p1, p2 []byte, player1Move, player2Move string) [][]byte {
	if player1Move == player2Move {
		return [][]byte{p1, p2} // draw
	} else if (player1Move == "rock" && player2Move == "scissors") ||
		(player1Move == "scissors" && player2Move == "paper") ||
		(player1Move == "paper" && player2Move == "rock") {
		return [][]byte{p1}
	}

	return [][]byte{p2}
}
