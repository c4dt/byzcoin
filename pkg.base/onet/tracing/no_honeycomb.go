package tracing

import (
	"os"

	"go.dedis.ch/onet/v3/log"
)

func init() {
	log.Info("Disabling Honeycomb")
	log.ErrFatal(os.Setenv("HONEYCOMB_API_KEY", ""))
}
