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
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"golang.org/x/xerrors"

	"github.com/urfave/cli"
	"go.dedis.ch/cothority/v3"
	_ "go.dedis.ch/cothority/v3/byzcoin"
	_ "go.dedis.ch/cothority/v3/byzcoin/contracts"
	_ "go.dedis.ch/cothority/v3/calypso"
	_ "go.dedis.ch/cothority/v3/personhood"
	_ "go.dedis.ch/cothority/v3/skipchain"
	status "go.dedis.ch/cothority/v3/status/service"
	"go.dedis.ch/kyber/v3/util/encoding"
	"go.dedis.ch/kyber/v3/util/key"
	"go.dedis.ch/onet/v3/app"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
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
			Name:    "server",
			Aliases: []string{"s"},
			Usage:   "Start server",
			Action:  runServer,
			/**
			  # ADDRESS_NODE should always be tls:// - tcp:// is insecure and should
			  # not be used.
			  - ADDRESS_NODE=tls://byzcoin.c4dt.org:7770
			  # ADDRESS_WS can be either http:// or https:// - for most of the use-cases
			  # you want this to be https://, so that secure webpages can access the node.
			  - ADDRESS_WS=https://byzcoin.c4dt.org:7771
			  # A short description of your node that will be visible to the outside.
			  - DESCRIPTION="New ByzCoin node"
			  # Only needed if ADDRESS_WS is https. Ignored if it is http.
			  - WS_SSL_CHAIN=fullchain.pem
			  - WS_SSL_KEY=privkey.pem
			  # ID of the byzcoin to follow - this corresponds to the DEDIS byzcoin.
			  - BYZCOIN_ID=9cc36071ccb902a1de7e0d21a2c176d73894b1cf88ae4cc2ba4c95cd76f474f3
			  # Where the data directory resides inside of the byzcoin container
			  - DATA_DIR=/byzcoin
			  # How much debugging output - 0 is none, 1 is important ones, 2 is
			  # interesting, 3 is detailed, 4 is lots of details, and 5 is too detailed for
			  # most purposes.
			  - DEBUG_LVL=2
			  # Whether to niceify the debug outputs. If you put this to `true`, you should
			  # have a black background in the terminal.
			  - DEBUG_COLOR=false
			  # Send the logging information to the c4dt logger. Optional, can be put to
			  # "" if not needed.
			  - GRAYLOG=graylog.c4dt.org:9001
			  # If set to "true", the binary will only update the configuration files
			  # and then quit.
			  - UPDATE_ONLY=false

			*/
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
					Name:  "ws-ssl-private, wsp",
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
				cli.BoolFlag{
					Name: "update-only, uo",
					Usage: "if true, will only update the ." +
						"toml files and then quit",
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

func runServer(c *cli.Context) error {
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
		if _, err := os.Stat(path.Join(dd, wsc)); err != nil {
			return xerrors.Errorf("error with ws-ssl-chain file: %+v", err)
		}
		conf.WebSocketTLSCertificate = app.CertificateURL(wsc)
	}

	if wsp := c.String("ws-ssl-private"); wsp != "" {
		if _, err := os.Stat(path.Join(dd, wsp)); err != nil {
			return xerrors.Errorf("error with ws-ssl-private file: %+v", err)
		}
		conf.WebSocketTLSCertificateKey = app.CertificateURL(wsp)
	}

	err := os.Mkdir(dd, 0770)
	if err != nil {
		return xerrors.Errorf("couldn't create config directory: %+v", err)
	}
	err = conf.Save(path.Join(dd, "private.toml"))
	if err != nil {
		return xerrors.Errorf("couldn't store private.toml: %+v", err)
	}

	if c.Bool("update-only") {
		log.Info("Quitting after update of configuration file")
		return nil
	}

	if bcIDStr := c.String("byzcoin-id"); bcIDStr != "" {
		bcID, err := hex.DecodeString(bcIDStr)
		if err != nil {
			return xerrors.Errorf("couldn't parse bcID: %+v", err)
		}
		// TODO: secure node to only accept this id
		log.Infof("Got BC-ID: %x", bcID)
	}
	return nil
}

func setup(c *cli.Context) error {
	if c.Bool("non-interactive") {
		host := c.String("host")
		port := c.Int("port")
		portStr := fmt.Sprintf("%v", port)

		serverBinding := network.NewAddress(network.TLS, net.JoinHostPort(host, portStr))
		kp := key.NewKeyPair(cothority.Suite)

		pub, _ := encoding.PointToStringHex(cothority.Suite, kp.Public)
		priv, _ := encoding.ScalarToStringHex(cothority.Suite, kp.Private)

		conf := &app.CothorityConfig{
			Suite:       cothority.Suite.String(),
			Public:      pub,
			Private:     priv,
			Address:     serverBinding,
			Description: c.String("description"),
			Services:    app.GenerateServiceKeyPairs(),
		}

		out := c.GlobalString("config")
		err := conf.Save(out)
		if err == nil {
			fmt.Fprintf(os.Stderr, "Wrote config file to %v\n", out)
		}

		// We are not going to write out the public.toml file here.
		// We don't because in the current use case for --non-interactive, which
		// is for containers to auto-generate configs on startup, the
		// roster (i.e. public IP addresses + public keys) will be generated
		// based on how Kubernetes does service discovery. Writing the public.toml
		// file based on the data we have here, would result in writing an invalid
		// public Address.

		// If we had written it, it would look like this:
		//  server := app.NewServerToml(cothority.Suite, kp.Public, conf.Address, conf.Description)
		//  group := app.NewGroupToml(server)
		//  group.Save(path.Join(dir, "public.toml"))

		return err
	}

	app.InteractiveConfig(cothority.Suite, DefaultName)
	return nil
}
