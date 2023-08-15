package keeper_test

import (
	"testing"

	"cosmossdk.io/core/genesis"
	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/facundomedica/rps"
	"github.com/facundomedica/rps/keeper"
)

type testFixture struct {
	ctx         sdk.Context
	k           keeper.Keeper
	msgServer   rps.MsgServer
	queryServer rps.QueryServer

	addrs []sdk.AccAddress
}

func initFixture(t *testing.T) *testFixture {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	key := storetypes.NewKVStoreKey(rps.ModuleName)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	storeService := runtime.NewKVStoreService(key)
	addrs := simtestutil.CreateIncrementalAccounts(3)

	k := keeper.NewKeeper(encCfg.Codec, addresscodec.NewBech32Codec("cosmos"), storeService, nil, addrs[0].String())

	source, err := genesis.SourceFromRawJSON([]byte(`{"game_id":[],"games":[],"move_commits":[],"move_reveals":[],"params":[{"key":"item","value":{"commit_timeout":"60","reveal_timeout":"60"}}]}`))
	require.NoError(t, err)

	err = k.Schema.InitGenesis(testCtx.Ctx, source)
	require.NoError(t, err)

	return &testFixture{
		ctx:         testCtx.Ctx,
		k:           k,
		msgServer:   keeper.NewMsgServerImpl(k),
		queryServer: keeper.NewQueryServerImpl(k),
		addrs:       addrs,
	}
}
