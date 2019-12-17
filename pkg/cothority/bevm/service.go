package bevm

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3/log"
)

func init() {
	// Ethereum starts goroutines for caching transactions, and never terminates them
	log.AddUserUninterestingGoroutine("go-ethereum/core.(*txSenderCacher).cache")

	log.ErrFatal(byzcoin.RegisterGlobalContract(ContractBEvmID, contractBEvmFromBytes))
}
