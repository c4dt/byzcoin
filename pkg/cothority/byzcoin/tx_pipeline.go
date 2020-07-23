package byzcoin

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

// rollupTxResult contains the aggregated response of the conodes to the
// rollupTx protocol.
type rollupTxResult struct {
	Txs           []ClientTransaction
	CommonVersion Version
}

// txProcessor is the interface that must be implemented. It is used in the
// stateful pipeline txPipeline that takes transactions and creates blocks.
type txProcessor interface {
	// ProcessTx attempts to apply the given tx to the input state and then
	// produce new state(s). If the new tx is too big to fit inside a new
	// state, the function will return more states. Where the older states
	// (low index) must be committed before the newer state (high index)
	// can be used. The function should only return error when there is a
	// catastrophic failure, if the transaction is refused then it should
	// not return error, but mark the transaction's Accept flag as false.
	ProcessTx(ClientTransaction, *txProcessorState) ([]*txProcessorState, error)
	// ProposeBlock should take the input state and propose the block. The
	// function should only return when a decision has been made regarding
	// the proposal.
	ProposeBlock(*txProcessorState) error
	// ProposeUpgradeBlock should create a barrier block between two Byzcoin
	// version so that future blocks will use the new version.
	ProposeUpgradeBlock(Version) error
	// GetLatestGoodState should return the latest state that the processor
	// trusts.
	GetLatestGoodState() *txProcessorState
	// GetBlockSize should return the maximum block size.
	GetBlockSize() int
	// GetInterval should return the block interval.
	GetInterval() time.Duration
	// Stop stops the txProcessor. Once it is called, the caller should not
	// expect the other functions in the interface to work as expected.
	Stop()
}

type txProcessorState struct {
	sst *stagingStateTrie

	// Below are changes that were made that led up to the state in sst
	// from the starting point.
	scs        StateChanges
	txs        TxResults
	txsSize    int
	newVersion Version
}

func (s *txProcessorState) size() int {
	if s.txsSize == 0 {
		body := &DataBody{TxResults: s.txs}
		payload, err := protobuf.Encode(body)
		if err != nil {
			return 0
		}
		s.txsSize = len(payload)
	}
	return s.txsSize
}

func (s *txProcessorState) reset() {
	s.scs = []StateChange{}
	s.txs = []TxResult{}
	s.txsSize = 0
}

// copy creates a shallow copy the state, we don't have the need for deep copy
// yet.
func (s *txProcessorState) copy() *txProcessorState {
	return &txProcessorState{
		s.sst.Clone(),
		append([]StateChange{}, s.scs...),
		append([]TxResult{}, s.txs...),
		s.txsSize,
		0,
	}
}

func (s txProcessorState) isEmpty() bool {
	return len(s.txs) == 0 && s.newVersion == 0
}

type defaultTxProcessor struct {
	*Service
	stopCollect chan bool
	scID        skipchain.SkipBlockID
	latest      *skipchain.SkipBlock
	sync.Mutex
}

func (s *defaultTxProcessor) ProcessTx(tx ClientTransaction, inState *txProcessorState) ([]*txProcessorState, error) {
	s.Lock()
	latest := s.latest
	s.Unlock()
	if latest == nil {
		return nil, xerrors.New("missing latest block in processor")
	}

	header, err := decodeBlockHeader(latest)
	if err != nil {
		return nil, xerrors.Errorf("decoding header: %v", err)
	}

	tx.Instructions.SetVersion(header.Version)

	scsOut, sstOut, err := s.processOneTx(inState.sst, tx, s.scID, header.Timestamp)

	// try to create a new state
	newState := func() *txProcessorState {
		if err != nil {
			return &txProcessorState{
				inState.sst,
				inState.scs,
				append(inState.txs, TxResult{tx, false}),
				0,
				0,
			}
		}
		return &txProcessorState{
			sstOut,
			append(inState.scs, scsOut...),
			append(inState.txs, TxResult{tx, true}),
			0,
			0,
		}
	}()

	// we're within the block size, so return one state
	if s.GetBlockSize() > newState.size() {
		return []*txProcessorState{newState}, nil
	}

	// if the new state is too big, we split it
	newStates := []*txProcessorState{inState.copy()}
	if err != nil {
		newStates = append(newStates, &txProcessorState{
			inState.sst,
			inState.scs,
			[]TxResult{{tx, false}},
			0,
			0,
		})
	} else {
		newStates = append(newStates, &txProcessorState{
			sstOut,
			scsOut,
			[]TxResult{{tx, true}},
			0,
			0,
		})
	}
	return newStates, nil
}

// ProposeBlock basically calls s.createNewBlock which might block. There is
// nothing we can do about it other than waiting for the timeout.
func (s *defaultTxProcessor) ProposeBlock(state *txProcessorState) error {
	config, err := state.sst.LoadConfig()
	if err != nil {
		return xerrors.Errorf("reading trie: %v", err)
	}
	_, err = s.createNewBlock(s.scID, &config.Roster, state.txs)
	return cothority.ErrorOrNil(err, "creating block")
}

func (s *defaultTxProcessor) ProposeUpgradeBlock(version Version) error {
	_, err := s.createUpgradeVersionBlock(s.scID, version)
	return cothority.ErrorOrNil(err, "creating block")
}

func (s *defaultTxProcessor) GetInterval() time.Duration {
	bcConfig, err := s.LoadConfig(s.scID)
	if err != nil {
		log.Error(s.ServerIdentity(), "couldn't get configuration - this is bad and probably "+
			"a problem with the database! ", err)
		return defaultInterval
	}
	return bcConfig.BlockInterval
}

func (s *defaultTxProcessor) GetLatestGoodState() *txProcessorState {
	st, err := s.getStateTrie(s.scID)
	if err != nil {
		// A good state must exist because we're working on a known
		// skipchain. If there is an error, then the database must've
		// failed, so there is nothing we can do to recover so we
		// panic.
		panic(fmt.Sprintf("failed to get a good state: %v", err))
	}
	return &txProcessorState{
		sst: st.MakeStagingStateTrie(),
	}
}

func (s *defaultTxProcessor) GetBlockSize() int {
	bcConfig, err := s.LoadConfig(s.scID)
	if err != nil {
		log.Error(s.ServerIdentity(), "couldn't get configuration - this is bad and probably "+
			"a problem with the database! ", err)
		return defaultMaxBlockSize
	}
	return bcConfig.MaxBlockSize
}

func (s *defaultTxProcessor) Stop() {
	close(s.stopCollect)
}

type txPipeline struct {
	ctxChan     chan ClientTransaction
	needUpgrade chan Version
	stopCollect chan bool
	wg          sync.WaitGroup
	processor   txProcessor
}

func newTxPipeline(s *Service, latest *skipchain.SkipBlock) *txPipeline {
	return &txPipeline{
		ctxChan:     make(chan ClientTransaction, 200),
		needUpgrade: make(chan Version, 1),
		stopCollect: make(chan bool),
		wg:          sync.WaitGroup{},
		processor: &defaultTxProcessor{
			Service:     s,
			stopCollect: make(chan bool),
			scID:        latest.SkipChainID(),
			latest:      latest,
			Mutex:       sync.Mutex{},
		},
	}
}

var maxTxHashes = 1000

func (p *txPipeline) createBlocks(newBlock chan *txProcessorState,
	done chan struct{}) {
	defer p.wg.Done()
	for {
		inState, ok := <-newBlock
		if !ok {
			break
		}
		if len(inState.txs) > 0 {
			// NOTE: ProposeBlock might block for a long time,
			// but there's nothing we can do about it at the moment
			// other than waiting for the timeout.
			err := p.processor.ProposeBlock(inState)
			if err != nil {
				log.Error("failed to propose block:", err)
				return
			}
		} else {
			// Create an upgrade block for the next version
			err := p.processor.ProposeUpgradeBlock(inState.newVersion)
			if err != nil {
				// Only log the error as it won't prevent normal blocks
				// to be created.
				log.Error("failed to upgrade", err)
			}
		}
		// TODO: tell the leaderLoop we're done and can take more states.
		done <- struct{}{}
	}
}

func (p *txPipeline) start(initialState *txProcessorState,
	stopSignal chan struct{}) {

	// always use the latest one when adding new
	currentState := txProcessorStates{initialState}
	var txHashes [][]byte

	newBlock := make(chan *txProcessorState, 1)
	done := make(chan struct{}, 1)
	p.wg.Add(1)
	go p.createBlocks(newBlock, done)

leaderLoop:
	for {
		select {
		case <-stopSignal:
			close(newBlock)
			break leaderLoop

		case <-done:
			if !currentState.isEmpty() {
				state := currentState.shift()
				select {
				case newBlock <- state:
				default:
					currentState.unshift(state)
				}
			}

		case version := <-p.needUpgrade:
			select {
			case latestState := <-newBlock:
				if latestState.newVersion == 0 {
					currentState.unshift(latestState)
				}
			default:
			}
			newVersion := &txProcessorState{newVersion: version,
				sst: currentState[0].sst}
			newBlock <- newVersion
			// Make sure that the <-p.ctxChan doesn't fetch the upgrade
			// instruction from newBlocks.
			if currentState.isEmpty() {
				currentState[0].newVersion = version
			}

		case tx := <-p.ctxChan:
			txh := tx.Instructions.HashWithSignatures()
			for _, txHash := range txHashes {
				if bytes.Compare(txHash, txh) == 0 {
					log.Lvl2("Got a duplicate transaction, ignoring it")
					continue leaderLoop
				}
			}
			txHashes = append(txHashes, txh)
			if len(txHashes) > maxTxHashes {
				txHashes = txHashes[len(txHashes)-maxTxHashes:]
			}

			latestState := currentState.shift()
			if latestState.isEmpty() {
				select {
				case nbState, ok := <-newBlock:
					if ok {
						latestState = nbState
					}
				default:
				}
			}

			// when processing, we take the latest state
			// (the last one) and then apply the new transaction to it
			newStates, err := p.processor.ProcessTx(tx, latestState)
			if err != nil {
				log.Error("processing transaction failed with error:", err)
				break
			}
			currentState.push(newStates...)

			// Try to send the first element of currentState to the newBlock.
			// If newBlock is already full, put the element back.
			first := currentState.shift()
			select {
			case newBlock <- first:
			default:
				currentState.unshift(first)
			}
		}
	}
	p.processor.Stop()
	p.wg.Wait()
}

type txProcessorStates []*txProcessorState

// shift generates the next input state that is used in
// ProposeBlock.
func (tps *txProcessorStates) shift() *txProcessorState {
	if len(*tps) == 1 {
		inState := (*tps)[0].copy()
		(*tps)[0].reset()
		return inState
	}
	inState, newTps := (*tps)[0], (*tps)[1:]
	*tps = newTps
	return inState
}

func (tps txProcessorStates) isEmpty() bool {
	return len(tps) == 1 && tps[0].isEmpty()
}

// push adds a new processorstate
func (tps *txProcessorStates) push(txs ...*txProcessorState) {
	for _, tx := range txs {
		i := len(*tps) - 1
		if (*tps)[i].isEmpty() {
			(*tps)[i] = tx
		} else {
			newTps := append(*tps, tx)
			*tps = newTps
		}
	}
}

func (tps *txProcessorStates) unshift(tx *txProcessorState) {
	if (*tps)[0].isEmpty() {
		(*tps)[0] = tx
	} else {
		newTps := append(txProcessorStates{tx}, *tps...)
		*tps = newTps
	}
}
