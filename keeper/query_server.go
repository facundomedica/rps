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
func (queryServer) Count(context.Context, *rps.QueryCountRequest) (*rps.QueryCountResponse, error) {
	panic("unimplemented")
}

// Games implements rps.QueryServer.
func (queryServer) Games(context.Context, *rps.QueryGamesRequest) (*rps.QueryGamesResponse, error) {
	panic("unimplemented")
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
