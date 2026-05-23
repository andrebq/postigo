package ioutil

import (
	"io"

	"github.com/hashicorp/yamux"
)

func MuxerConfig() *yamux.Config {
	opt := yamux.DefaultConfig()
	opt.LogOutput = io.Discard
	return opt
}
