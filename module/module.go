package module

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"

	"cosmossdk.io/core/appmodule"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/facundomedica/rps"
	"github.com/facundomedica/rps/keeper"
	"github.com/facundomedica/rps/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ConsensusVersion defines the current module consensus version.
const ConsensusVersion = 1

type AppModule struct {
	appmodule.HasGenesis

	keeper keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(keeper keeper.Keeper) AppModule {
	return AppModule{
		// using the keeper's schema as the genesis schema, this means all state must use collections
		HasGenesis: keeper.Schema,
		keeper:     keeper,
	}
}

func NewAppModuleBasic(m AppModule) module.AppModuleBasic {
	return module.CoreAppModuleBasicAdaptor(m.Name(), m)
}

// Name returns the rps module's name.
func (AppModule) Name() string { return rps.ModuleName }

// RegisterLegacyAminoCodec registers the rps module's types on the LegacyAmino codec.
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	rps.RegisterLegacyAminoCodec(cdc)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the rps module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := rps.RegisterQueryHandlerClient(context.Background(), mux, rps.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers interfaces and implementations of the rps module.
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	rps.RegisterInterfaces(registry)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	rps.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	rps.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))

	// Register in place module state migration migrations
	// m := keeper.NewMigrator(am.keeper)
	// if err := cfg.RegisterMigration(rps.ModuleName, 1, m.Migrate1to2); err != nil {
	// 	panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", rps.ModuleName, err))
	// }
}

func (AppModule) GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  rps.ModuleName,
		Args: cobra.ExactArgs(1),
		RunE: client.ValidateCmd,
	}

	cmd.AddCommand(
		newGameCmd(),
		commitMoveCmd(),
		revealMoveCmd(),
	)
	return cmd
}

func newGameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new-game [move] [entry_fee]",
		Short: "Create a new game",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			playerAddr := clientCtx.GetFromAddress()

			move := args[0]

			if !utils.MoveIsValid(move) {
				return errors.New("invalid move")
			}

			salt, err := salt(32)
			if err != nil {
				return err
			}

			commit := utils.CalculateCommitment(move, salt)

			fee, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			msg := &rps.MsgNewGame{
				Player:   playerAddr.String(),
				Commit:   commit,
				EntryFee: fee,
			}

			cmd.Println("Copy your salt for the reveal stage:", salt)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func commitMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit-move [game_id] [move]",
		Short: "Enter a game",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			playerAddr := clientCtx.GetFromAddress()

			gameID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			move := args[1]

			if !utils.MoveIsValid(move) {
				return errors.New("invalid move")
			}

			salt, err := salt(32)
			if err != nil {
				return err
			}

			commit := utils.CalculateCommitment(move, salt)

			msg := &rps.MsgCommitMove{
				Player: playerAddr.String(),
				GameId: gameID,
				Commit: commit,
			}

			cmd.Println("Copy your salt for the reveal stage:", salt)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func revealMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reveal-move [game_id] [move] [salt]",
		Short: "Reveal your move",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			playerAddr := clientCtx.GetFromAddress()

			gameID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			move := args[1]

			if !utils.MoveIsValid(move) {
				return errors.New("invalid move")
			}

			salt := args[2]

			msg := &rps.MsgRevealMove{
				Player: playerAddr.String(),
				GameId: gameID,
				Move:   move,
				Salt:   salt,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func salt(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
