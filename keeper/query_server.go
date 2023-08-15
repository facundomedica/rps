package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/facundomedica/rps"
)

var _ rps.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the module QueryServer.
func NewQueryServerImpl(k Keeper) rps.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

// Count implements rps.QueryServer.
func (qs queryServer) Count(ctx context.Context, _ *rps.QueryCountRequest) (*rps.QueryCountResponse, error) {
	count, err := qs.k.GameID.Next(ctx)
	if err != nil {
		return nil, err
	}

	return &rps.QueryCountResponse{Count: count}, nil
}

// Games implements rps.QueryServer.
func (qs queryServer) Games(ctx context.Context, _ *rps.QueryGamesRequest) (*rps.QueryGamesResponse, error) {
	res := &rps.QueryGamesResponse{Games: []rps.Game{}}

	err := qs.k.Games.Walk(ctx, nil, func(key uint64, game rps.Game) (bool, error) {
		game.Id = key
		res.Games = append(res.Games, game)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Params defines the handler for the Query/Params RPC method.
func (qs queryServer) Params(ctx context.Context, req *rps.QueryParamsRequest) (*rps.QueryParamsResponse, error) {
	params, err := qs.k.Params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &rps.QueryParamsResponse{Params: rps.Params{}}, nil
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	return &rps.QueryParamsResponse{Params: params}, nil
}
