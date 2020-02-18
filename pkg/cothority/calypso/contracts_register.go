package calypso

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3/log"
)

var readMakeAttrInterpreter = make([]makeAttrInterpreterWrapper, 0)

// makeAttrInterpreterWrapper holds the data needed to register a
// MakeAttrInterpreter.
type makeAttrInterpreterWrapper struct {
	// name is the corresponding name of the custom attribute.
	name string
	// interpreter is the function producing the interepreter for the given name.
	// We are using a callback to have access to the instance's context.
	interpreter func(c ContractWrite, rst byzcoin.ReadOnlyStateTrie,
		inst byzcoin.Instruction) func(string) error
}

func init() {
	log.ErrFatal(byzcoin.RegisterGlobalContract(ContractWriteID,
		contractWriteFromBytes))
	log.ErrFatal(byzcoin.RegisterGlobalContract(ContractReadID,
		contractReadFromBytes))
	log.ErrFatal(byzcoin.RegisterGlobalContract(ContractLongTermSecretID,
		contractLTSFromBytes))
}
