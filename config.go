package zaprpc

import (
	"crypto/tls"

	"github.com/quic-go/quic-go"
)

type ZapConfig struct {
	tlsConfig       *tls.Config
	quicConfig      *quic.Config
	transportConfig *quic.Transport
}
