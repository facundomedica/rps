package rps

import "cosmossdk.io/collections"

const ModuleName = "rps"

var (
	ParamsKey     = collections.NewPrefix(0)
	GameIDKey     = collections.NewPrefix(1)
	GamesKey      = collections.NewPrefix(2)
	MoveCommitKey = collections.NewPrefix(3)
	MoveRevealKey = collections.NewPrefix(4)
)
