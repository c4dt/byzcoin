package byzcoin

import (
	"bytes"
	"fmt"
	"go.dedis.ch/cothority/v3/darc/expression"
	"go.dedis.ch/protobuf"

	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/darc"
	"golang.org/x/xerrors"
)

// ContractDarcID denotes a secure version of the DARC contract. We
// provide two forms of security. The first is "restricted evolution", where
// the evolve command only allows changes to existing rules, it is not allowed
// to add new rules. There exists an additional command "evolve_unrestricted"
// that allows authorised users to change the rules arbitrarily. Our second
// form of security is "controlled spawn", where the rules of the secure darcs
// spawned using this contract are subject to some restrictions, e.g., the new
// rules must not contain spawn:inseucre_darc. While this contract may be
// useful in a lot of scenarios, it is possible to have even more control by
// writing new DARC contracts for the intended application.
const ContractDarcID = "darc"

type contractSecureDarc struct {
	BasicContract
	darc.Darc
	contracts ReadOnlyContractRegistry
}

var _ Contract = (*contractSecureDarc)(nil)

const cmdDarcEvolveUnrestriction = "evolve_unrestricted"
const cmdDarcEvolve = "evolve"

func contractSecureDarcFromBytes(in []byte) (Contract, error) {
	d, err := darc.NewFromProtobuf(in)
	if err != nil {
		return nil, xerrors.Errorf("darc decoding: %v", err)
	}
	c := &contractSecureDarc{Darc: *d}
	return c, nil
}

// SetRegistry keeps the reference of the contract registry.
func (c *contractSecureDarc) SetRegistry(r ReadOnlyContractRegistry) {
	c.contracts = r
}

// VerifyDeferredInstruction does the same as the standard VerifyInstruction
// method in the diferrence that it does not take into account the counters. We
// need the Darc contract to opt in for deferred transaction because it is used
// by default when spawning new contracts.
func (c *contractSecureDarc) VerifyDeferredInstruction(rst ReadOnlyStateTrie, inst Instruction, ctxHash []byte) error {
	err := inst.VerifyWithOption(rst, ctxHash, &VerificationOptions{IgnoreCounters: true})
	return cothority.ErrorOrNil(err, "instruction verification")
}

func (c *contractSecureDarc) Spawn(rst ReadOnlyStateTrie, inst Instruction, coins []Coin) (sc []StateChange, cout []Coin, err error) {
	cout = coins

	if inst.Spawn.ContractID == ContractDarcID {
		darcBuf := inst.Spawn.Args.Search("darc")
		d, err := darc.NewFromProtobuf(darcBuf)
		if err != nil {
			return nil, nil, xerrors.Errorf("given DARC could not be decoded: %v", err)
		}
		if d.Version != 0 {
			return nil, nil, xerrors.New("DARC version must start at 0")
		}

		id := d.GetBaseID()

		// Here is an example hard-coded constraint for spawning DARCs.
		// If the constraint needs to be dynamic, then it is
		// recommended to create a new contract that contains mappings
		// of roles -> identities, and roles -> whitelist of rules.
		// Then modify this contract to check the whitelist.
		if d.Rules.Contains("spawn:insecure_darc") {
			return nil, nil, xerrors.New("a secure DARC is not allowed to spawn an insecure DARC")
		}

		return []StateChange{
			NewStateChange(Create, NewInstanceID(id), ContractDarcID, darcBuf, id),
		}, coins, nil
	}

	// If we got here this is a spawn:xxx in order to spawn
	// a new instance of contract xxx, so do that.

	if c.contracts == nil {
		return nil, nil, xerrors.New("contracts registry is missing due to bad initialization")
	}

	cfact, found := c.contracts.Search(inst.Spawn.ContractID)
	if !found {
		return nil, nil, xerrors.New("couldn't find this contract type: " + inst.Spawn.ContractID)
	}

	// Pass nil into the contract factory here because this instance does not exist yet.
	// So the factory will make a zero-value instance, and then calling Spawn on it
	// will give it a chance to encode it's zero state and emit one or more StateChanges to put itself
	// into the trie.
	c2, err := cfact(nil)
	if err != nil {
		return nil, nil, xerrors.Errorf("could not spawn new zero instance: %v", err)
	}
	if cwr, ok := c2.(ContractWithRegistry); ok {
		cwr.SetRegistry(c.contracts)
	}

	scs, coins, err := c2.Spawn(rst, inst, coins)
	return scs, coins, cothority.ErrorOrNil(err, "spawn instance")
}

func (c *contractSecureDarc) Invoke(rst ReadOnlyStateTrie, inst Instruction, coins []Coin) ([]StateChange, []Coin, error) {
	switch inst.Invoke.Command {
	case cmdDarcEvolve:
		var darcID darc.ID
		_, _, _, darcID, err := rst.GetValues(inst.InstanceID.Slice())
		if err != nil {
			return nil, nil, xerrors.Errorf("reading trie: %v", err)
		}

		darcBuf := inst.Invoke.Args.Search("darc")
		newD, err := darc.NewFromProtobuf(darcBuf)
		if err != nil {
			return nil, nil, xerrors.Errorf("darc encoding: %v", err)
		}
		oldD, err := rst.LoadDarc(darcID)
		if err != nil {
			return nil, nil, xerrors.Errorf("darc from trie: %v", err)
		}
		// do not allow modification of evolve_unrestricted
		if isChangingEvolveUnrestricted(oldD, newD) {
			return nil, nil, xerrors.New("the evolve command is not allowed to change the the evolve_unrestricted rule")
		}
		if err := newD.SanityCheck(oldD); err != nil {
			return nil, nil, xerrors.Errorf("sanity check: %v", err)
		}
		// use the subset rule if it's not a genesis Darc
		_, _, _, genesisDarcID, err := GetValueContract(rst, NewInstanceID(nil).Slice())
		if err != nil {
			return nil, nil, xerrors.Errorf("getting contract: %v", err)
		}
		if !genesisDarcID.Equal(oldD.GetBaseID()) {
			if !newD.Rules.IsSubset(oldD.Rules) {
				return nil, nil, xerrors.New("rules in the new version must be a subset of the previous version")
			}
		}
		return []StateChange{
			NewStateChange(Update, inst.InstanceID, ContractDarcID, darcBuf, darcID),
		}, coins, nil
	case cmdDarcEvolveUnrestriction:
		var darcID darc.ID
		_, _, _, darcID, err := rst.GetValues(inst.InstanceID.Slice())
		if err != nil {
			return nil, nil, xerrors.Errorf("reading trie: %v", err)
		}

		darcBuf := inst.Invoke.Args.Search("darc")
		newD, err := darc.NewFromProtobuf(darcBuf)
		if err != nil {
			return nil, nil, xerrors.Errorf("encoding darc: %v", err)
		}
		oldD, err := rst.LoadDarc(darcID)
		if err != nil {
			return nil, nil, xerrors.Errorf("darc from trie: %v", err)
		}
		if err := newD.SanityCheck(oldD); err != nil {
			return nil, nil, xerrors.Errorf("sanity check: %v", err)
		}
		return []StateChange{
			NewStateChange(Update, inst.InstanceID, ContractDarcID, darcBuf, darcID),
		}, coins, nil
	default:
		return nil, nil, xerrors.New("invalid command: " + inst.Invoke.Command)
	}
}

func isChangingEvolveUnrestricted(oldD *darc.Darc, newD *darc.Darc) bool {
	oldExpr := oldD.Rules.Get(darc.Action("invoke:" + ContractDarcID + "." + cmdDarcEvolveUnrestriction))
	newExpr := newD.Rules.Get(darc.Action("invoke:" + ContractDarcID + "." + cmdDarcEvolveUnrestriction))
	if len(oldExpr) == 0 && len(newExpr) == 0 {
		return false
	}
	if bytes.Equal(oldExpr, newExpr) {
		return false
	}
	return true
}

// ContractDarcSpawnInstructions creates one or more Instructions for
// spawning a darc.
func ContractDarcSpawnInstructions(spawnerID darc.ID,
	darcs ...darc.Darc) (instr []Instruction, err error) {
	for _, d := range darcs {
		darcBuf, err := d.ToProto()
		if err != nil {
			return nil, xerrors.Errorf("encoding darc: %v", err)
		}
		instr = append(instr, Instruction{
			InstanceID: NewInstanceID(spawnerID),
			Spawn: &Spawn{
				ContractID: ContractDarcID,
				Args: Arguments{{
					Name:  "darc",
					Value: darcBuf,
				}},
			},
		})
	}
	return
}

// ContractDarcGetFromID retrieves a darc from the byzcoin instance given in
// the client. It returns any error encountered while getting the darc.
func ContractDarcGetFromID(cl *Client, id darc.ID) (d darc.Darc, err error) {
	resp, err := cl.GetProof(id)
	if err != nil {
		return d, xerrors.Errorf("while getting proof for darc: %v", err)
	}
	val, cid, _, err := resp.Proof.Get(id)
	if err != nil {
		return d, xerrors.Errorf("couldn't get darc-id from proof: %v", err)
	}
	if cid != ContractDarcID {
		return d, xerrors.Errorf("this InstanceID points to a '%s', "+
			"not a darc", cid)
	}
	if err = protobuf.Decode(val, &d); err != nil {
		return d, xerrors.Errorf("couldn't decode darc: %v", err)
	}
	return
}

// ContractDarcAssertActions makes sure that the given actions include the given
// identity. If not, it will try to update the darc to actually have these
// rules present. This requires that the darc is of type "evolve_unrestricted"
func ContractDarcAssertActions(cl *Client, darcID darc.ID,
	actions []darc.Action, id darc.Identity, signer darc.Signer) error {
	d, err := ContractDarcGetFromID(cl, darcID)
	if err != nil {
		return xerrors.Errorf("couldn't get darc from byzcoin: %v", err)
	}
	evUnres := d.Rules.Get("invoke:darc." + cmdDarcEvolveUnrestriction)
	if evUnres == nil {
		return xerrors.Errorf("this is not a darc with a '%s' rule", cmdDarcEvolveUnrestriction)
	}
	dNew := d.Copy()
	if err := dNew.EvolveFrom(&d); err != nil {
		return xerrors.Errorf("couldn't create evolving darc: %v", err)
	}

	updated := false
	for _, a := range actions {
		rule := d.Rules.Get(a)
		if rule != nil {
			ok, err := expression.DefaultParser(rule, id.String())
			if err == nil && ok {
				continue
			}
			rule = append(rule, fmt.Sprintf(" | %s", id.String())...)
			if err := dNew.Rules.UpdateRule(a, rule); err != nil {
				return xerrors.Errorf("couldn't update rule: %v", err)
			}
		} else {
			if err := dNew.Rules.AddRule(a,
				expression.Expr(id.String())); err != nil {
				return xerrors.Errorf("couldn't add rule: %v", err)
			}
		}

		updated = true
	}

	if updated {
		dNewBuf, err := dNew.ToProto()
		if err != nil {
			return xerrors.Errorf("couldn't encode new darc: %v", err)
		}
		ctx, err := cl.CreateTransaction(Instruction{
			InstanceID: NewInstanceID(dNew.GetBaseID()),
			Invoke: &Invoke{
				ContractID: ContractDarcID,
				Command:    cmdDarcEvolveUnrestriction,
				Args: Arguments{
					{Name: "darc",
						Value: dNewBuf},
				},
			},
		})
		if err != nil{
			return xerrors.Errorf("couldn't create new transaction: %v", err)
		}
		if err := cl.SignTransaction(ctx, signer); err != nil{
			return xerrors.Errorf("while signing transaction: %v", err)
		}
		if _, err := cl.AddTransactionAndWait(ctx, 10); err != nil{
			return xerrors.Errorf("while submitting transaction: %v", err)
		}
	}
	return nil
}
