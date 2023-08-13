package rps

import "cosmossdk.io/collections"

const ModuleName = "rps"

var (
	ParamsKey  = collections.NewPrefix(0)
	CounterKey = collections.NewPrefix(1)
)
