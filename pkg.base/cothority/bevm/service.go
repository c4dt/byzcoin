package bevm

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3/log"
)

func init() {
	log.ErrFatal(byzcoin.RegisterGlobalContract(ContractBEvmID,
		contractBEvmFromBytes))
}
