// Conode is the main binary for running a Cothority server.
// A conode can participate in various distributed protocols using the
// *onet* library as a network and overlay library and the *kyber*
// library for all cryptographic primitives.
// Basically, you first need to setup a config file for the server by using:
//
//  ./conode setup
//
// Then you can launch the daemon with:
//
//  ./conode
//
// Services need to be imported to be available when the conode is
// running.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"time"

	"go.dedis.ch/cothority/v3/skipchain"

	"github.com/BurntSushi/toml"
	"golang.org/x/xerrors"

	"github.com/urfave/cli"
	"go.dedis.ch/cothority/v3"
	_ "go.dedis.ch/cothority/v3/bevm"
	_ "go.dedis.ch/cothority/v3/byzcoin"
	_ "go.dedis.ch/cothority/v3/byzcoin/contracts"
	_ "go.dedis.ch/cothority/v3/calypso"
	_ "go.dedis.ch/cothority/v3/personhood/contracts"
	_ "go.dedis.ch/cothority/v3/skipchain"
	status "go.dedis.ch/cothority/v3/status/service"
	"go.dedis.ch/kyber/v3/util/encoding"
	"go.dedis.ch/kyber/v3/util/key"
	"go.dedis.ch/onet/v3/app"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	_ "go.dedis.ch/onet/v3/tracing/service"
)

const (
	// DefaultName is the name of the binary we produce and is used to create a directory
	// folder with this name
	DefaultName = "byzcoin"
)

var gitTag = ""

func main() {
	cliApp := cli.NewApp()
	cliApp.Name = DefaultName
	cliApp.Usage = "Run a ByzCoin node"
	if gitTag == "" {
		cliApp.Version = "unknown"
	} else {
		cliApp.Version = gitTag
	}
	status.Version = cliApp.Version

	cliApp.Commands = []cli.Command{
		{
			Name:    "config",
			Aliases: []string{"s"},
			Usage:   "Create configuration",
			Action:  configure,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "address-node, addr-n",
					Usage: "defines the name:port of the node-to-node" +
						" communication",
				},
				cli.StringFlag{
					Name: "address-ws, addr-ws",
					Usage: "defines the name:port of the websocket" +
						" communication",
				},
				cli.StringFlag{
					Name:  "description, desc",
					Usage: "a short description of the node, <= 32 chars",
				},
				cli.StringFlag{
					Name:  "ws-ssl-chain, wsc",
					Usage: "the fullchain.pem file for the websocket port",
				},
				cli.StringFlag{
					Name:  "ws-ssl-key, wsk",
					Usage: "the privkey.pem file for the websocket port",
				},
				cli.StringFlag{
					Name: "byzcoin-id, bcid",
					Usage: "the hex representation of the byzcoin-id to" +
						" connect to",
				},
				cli.StringFlag{
					Name:  "data-dir, dd",
					Usage: "where the configuration files should be stored",
					Value: "./bc-data",
				},
			},
		},
		{
			Name:      "show",
			Aliases:   []string{"s"},
			Usage:     "Show configuration",
			Action:    showConfig,
			ArgsUsage: "data-dir",
		},
		{
			Name:      "run",
			Aliases:   []string{"r"},
			Usage:     "Run the node",
			Action:    run,
			ArgsUsage: "data-dir",
		},
		{
			Name:      "proxy",
			Aliases:   []string{"p"},
			Usage:     "Act as a proxy to the real nodes",
			Action:    runProxy,
			ArgsUsage: "db-file",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "portWS, p",
					Value: 7771,
					Usage: "port-number of the websocket",
				},
				cli.StringFlag{
					Name:  "update, u",
					Value: "1m",
					Usage: "update interval",
				},
				cli.BoolFlag{
					Name:  "complete",
					Usage: "whether to fetch the complete chain",
				},
			},
		},
	}
	cliApp.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "debug, d",
			Value: 0,
			Usage: "debug-level: 1 for terse, 5 for maximal",
		},
	}
	cliApp.Before = func(c *cli.Context) error {
		log.SetDebugVisible(c.Int("debug"))
		return nil
	}

	// Do not allow conode to run when built in 32-bit mode.
	// The dedis/protobuf package is the origin of this limit.
	// Instead of getting the error later from protobuf and being
	// confused, just make it totally clear up-front.
	var i int
	iType := reflect.TypeOf(i)
	if iType.Size() < 8 {
		log.ErrFatal(errors.New("conode cannot run when built in 32-bit mode"))
	}

	err := cliApp.Run(os.Args)
	log.ErrFatal(err)
}

// configure creates a new private.toml from the command-line arguments.
// It has some sensible defaults for some of the arguments.
func configure(c *cli.Context) error {
	conf := &app.CothorityConfig{Suite: cothority.Suite.String()}
	dd := c.String("data-dir")
	fn := path.Join(dd, "private.toml")
	if ccFile, err := ioutil.ReadFile(fn); err == nil {
		log.Info("Loading configuration from private.toml")
		_, err = toml.Decode(string(ccFile), conf)
		if err != nil {
			return xerrors.Errorf("couldn't parse private.toml: %+v", err)
		}
	} else {
		log.Info("Creating keypair")
		kp := key.NewKeyPair(cothority.Suite)
		conf.Public, err = encoding.PointToStringHex(cothority.Suite, kp.Public)
		if err != nil {
			return xerrors.Errorf("couldn't get public buffer: %+v", err)
		}
		conf.Private, err = encoding.ScalarToStringHex(cothority.Suite, kp.Private)
		if err != nil {
			return xerrors.Errorf("couldn't get private buffer: %+v", err)
		}
		conf.Services = app.GenerateServiceKeyPairs()
		conf.Description = "New ByzCoin node"
	}

	if node := c.String("address-node"); node != "" {
		if !strings.HasPrefix(node, "tls://") {
			return xerrors.New("node address must start with tls://")
		}
		conf.Address = network.Address(node)
	}

	if node := c.String("address-ws"); node != "" {
		if http, _ := regexp.MatchString("https?://", node); !http {
			return xerrors.New("websocket address must start with https:// or" +
				" http://")
		}
		conf.URL = node
	}

	if desc := c.String("description"); desc != "" {
		if len(desc) > 32 {
			return xerrors.New("description length cannot be longer than 32")
		}
		conf.Description = desc
	}

	if wsc := c.String("ws-ssl-chain"); wsc != "" {
		if _, err := os.Stat(wsc); err != nil {
			return xerrors.Errorf("error with ws-ssl-chain file: %+v", err)
		}
		conf.WebSocketTLSCertificate = app.CertificateURL(wsc)
	}

	if wsk := c.String("ws-ssl-key"); wsk != "" {
		if _, err := os.Stat(wsk); err != nil {
			return xerrors.Errorf("error with ws-ssl-key file: %+v", err)
		}
		conf.WebSocketTLSCertificateKey = app.CertificateURL(wsk)
	}

	err := conf.Save(path.Join(dd, "private.toml"))
	if err != nil {
		return xerrors.Errorf("couldn't store private.toml: %+v", err)
	}

	siToml := &app.ServerToml{
		Address:     conf.Address,
		Suite:       conf.Suite,
		Public:      conf.Public,
		Description: conf.Description,
		Services:    make(map[string]app.ServerServiceConfig),
		URL:         conf.URL,
	}
	for name, serviceConfig := range conf.Services {
		siToml.Services[name] = app.ServerServiceConfig{
			Suite:  serviceConfig.Suite,
			Public: serviceConfig.Public,
		}
	}

	if err := ioutil.WriteFile(path.Join(dd, "public.toml"),
		[]byte(siToml.String()), 0644); err != nil {
		return xerrors.Errorf("couldn't write public.toml: %+v", err)
	}

	if bcIDStr := c.String("byzcoin-id"); bcIDStr != "" {
		bcID, err := hex.DecodeString(bcIDStr)
		if err != nil {
			return xerrors.Errorf("couldn't parse bcID: %+v", err)
		}

		_, server, err := app.ParseCothority(path.Join(dd, "private.toml"))
		if err != nil {
			return xerrors.Errorf("couldn't load config: %+v", err)
		}
		go server.Start()
		server.WaitStartup()

		ss := server.Service(skipchain.ServiceName).(*skipchain.Service)
		ss.Storage.FollowIDs = []skipchain.SkipBlockID{bcID}
		// Abusing AddClientKey to call ss.save(), which is a private method.
		kp := key.NewKeyPair(cothority.Suite)
		ss.AddClientKey(kp.Public)

		if err = server.Stop(); err != nil {
			return xerrors.Errorf("couldn't stop server: %+v", err)
		}
	}

	return nil
}

// showConfig prints the private.toml in a human readable form.
func showConfig(c *cli.Context) error {
	if c.NArg() != 1 {
		return xerrors.New("Please give data-dir")
	}

	_, server, err := app.ParseCothority(path.Join(c.Args().First(), "private.toml"))
	if err != nil {
		return xerrors.Errorf("couldn't load config: %+v", err)
	}
	go server.Start()
	server.WaitStartup()

	ss := server.Service(skipchain.ServiceName).(*skipchain.Service)
	if len(ss.Storage.FollowIDs) == 0 {
		log.Info("No IDs followed")
	}
	for _, id := range ss.Storage.FollowIDs {
		log.Infof("Following ID: %x", id)
	}

	if err = server.Stop(); err != nil {
		return xerrors.Errorf("couldn't stop server: %+v", err)
	}

	return nil
}

// run starts the node as a server, given a private.toml.
func run(c *cli.Context) error {
	if c.NArg() != 1 {
		return xerrors.New("Please give data-dir")
	}

	_, server, err := app.ParseCothority(path.Join(c.Args().First(), "private.toml"))
	if err != nil {
		return xerrors.Errorf("couldn't load config: %+v", err)
	}
	server.Start()
	return xerrors.New("server stopped unexpectedly")
}

// runProxy starts only a proxy that is not part of the network,
// but merely fetches new blocks from the other nodes.
// You can use this to have a fast node locally.
// All security assumptions are still given,
// as the forward-links will prove that the new blocks are correct.
// It doesn't need to have a private.toml,
// but merely uses a db with at least one chain defined.
func runProxy(c *cli.Context) error {
	if c.NArg() != 1 {
		return xerrors.New("Please give database")
	}

	dur, err := time.ParseDuration(c.String("update"))
	if err != nil {
		return xerrors.Errorf("couldn't parse update: %v", err)
	}

	// Create a dummy ServerIdentity for the server to run.
	addr := network.Address(fmt.Sprintf("tls://localhost:%d",
		c.Int("portWS")-1))
	privKey := cothority.Suite.Scalar().One()
	pubKey := cothority.Suite.Point().Mul(privKey, nil)
	si := network.NewServerIdentity(pubKey, addr)
	si.SetPrivate(privKey)
	services := make(map[string]app.ServiceConfig)
	for _, name := range onet.ServiceFactory.RegisteredServiceNames() {
		serviceSuite := onet.ServiceFactory.Suite(name)
		if serviceSuite != nil {
			kp := key.NewKeyPair(serviceSuite)
			var pubBuf bytes.Buffer
			if err := encoding.WriteHexPoint(serviceSuite, &pubBuf,
				kp.Public); err != nil{
				return xerrors.Errorf("couldn't marshal point: %v", err)
			}
			services[name] = app.ServiceConfig{Suite: serviceSuite.String(),
				Public: pubBuf.String()}
			si.ServiceIdentities = append(si.ServiceIdentities,
				network.NewServiceIdentity(name, serviceSuite, kp.Public,
					kp.Private))
		}
	}

	serverToml := app.NewServerToml(cothority.Suite, pubKey, addr, "Proxy Server",
		services)
	if serverToml == nil{
		return xerrors.New("unknown error when creating NewServerToml")
	}
	publicToml := app.NewGroupToml(serverToml)
	log.Infof("public.toml:\n%s", publicToml.String())

	// HACK: as we cannot give the db-name to the onet.Server structure,
	// this creates a link in the /tmp-directory that points to the db given
	// on the command-line.
	pub, _ := pubKey.MarshalBinary()
	dbName := fmt.Sprintf("%x.db", sha256.Sum256(pub))
	if err := os.Setenv("CONODE_SERVICE_PATH", "/tmp"); err != nil {
		return xerrors.Errorf("couldn't set CONODE_SERVICE_PATH: %v", err)
	}

	dbPath := path.Join("/tmp", dbName)
	// Ignore the error of the link not found
	_ = os.Remove(dbPath)
	argFullPath := c.Args().First()
	if !path.IsAbs(argFullPath) {
		wd, err := os.Getwd()
		if err != nil {
			return xerrors.Errorf("couldn't get working directory: %v", err)
		}
		argFullPath = path.Join(wd, argFullPath)
	}
	if err := os.Symlink(argFullPath, dbPath); err != nil {
		return xerrors.Errorf("couldn't create a symlink: %v", err)
	}

	// Finally get the server.
	server := onet.NewServerTCP(si, cothority.Suite)

	// Update the skipchains on a regular basis.
	// This will also take care to update the global state.
	go func() {
		server.WaitStartup()
		byzcoinService := server.Service(byzcoin.ServiceName).(*byzcoin.Service)

		// Fetch all missing blocks if 'complete' is given.
		if c.Bool("complete") {
			err := updateSkipchains(server.Service(skipchain.ServiceName).(*skipchain.Service))
			if err != nil {
				log.Errorf("Couldn't update all blocks: %+v", err)
			}
		}

		// To get the latest blocks, restart the service.
		// This might create new holes in the chain if there are too many
		// blocks created between two calls.
		for {
			time.Sleep(dur)
			log.Info("Fetching new blocks")
			byzcoinService.TestClose()
			if err := byzcoinService.TestRestart(); err != nil {
				log.Fatalf("Couldn't restart service: %+v", err)
			}
		}
	}()

	server.Start()
	return xerrors.New("server stopped unexpectedly")
}

// updateSkipchains does what skipchain.Service.
// DoSync doesn't do: it fills all holes in the skipchain.
func updateSkipchains(scs *skipchain.Service) error {
	log.Info("Searching and downloading blocks for all chains")
	scIDs, err := scs.GetDB().GetSkipchains()
	if err != nil {
		return xerrors.Errorf("couldn't fetch all skipchain-IDs: %v", err)
	} else {
	chains:
		for sc, latest := range scIDs {
			scID := skipchain.SkipBlockID(sc)
			log.Lvlf1("Checking chain %x up to block %d", scID[:], latest.Index)
			current := scs.GetDB().GetByID(scID)
			cl := skipchain.NewClient()
			for {
				if current.Index%10000 == 0 {
					log.Lvlf1("Scanning block %d/%d", current.Index,
						latest.Index)
				}
				var next *skipchain.SkipBlock
				if len(current.ForwardLink) == 0 {
					// latest might already be outdated once we get here,
					// in the case where the skipchain-service found a later
					// block.
					latest, err = scs.GetDB().GetLatestByID(scID)
					if err != nil {
						return xerrors.Errorf(
							"couldn't get latest block: %v", err)
					}
					if current.Hash.Equal(latest.Hash) {
						log.Lvlf1("Done with chain %x at block %d",
							scID[:], current.Index)
						return nil
					}
					log.Lvlf1("Found premature end of chain - downloading hole")
				} else {
					next = scs.GetDB().GetByID(current.ForwardLink[0].To)
				}

				if next != nil {
					current = next
				} else {
					log.Lvlf1("Downloading blocks at index %d", current.Index)
					blocks, err := cl.GetUpdateChainLevel(latest.Roster,
						current.Hash, 1, 1000)
					if err != nil {
						return xerrors.Errorf(
							"error while getting blocks: %v", err)
					}
					if len(blocks) < 2 {
						log.Warn("Couldn't get new blocks")
						continue chains
					}
					log.Lvlf1("Got %d new blocks", len(blocks))
					_, err = scs.GetDB().StoreBlocks(blocks)
					if err != nil {
						return xerrors.Errorf("couldn't store blocks: %v", err)
					}
					for _, sb := range blocks[1:] {
						if sb.Index == current.Index+1 {
							current = sb
						} else {
							break
						}
					}
				}
			}
		}
	}
	return nil
}
