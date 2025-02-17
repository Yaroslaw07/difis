package tcp

import (
	"testing"

	"github.com/Yaroslaw07/difis/pkg/p2p"
	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	opts := TCPTransportOpts{
		ListenAddr:    ":3000",
		Decoder:       p2p.DefaultDecoder{},
		HandshakeFunc: p2p.NOPHandshakeFunc,
	}
	tr := NewTCPTransport(opts)

	assert.Equal(t, ":3000", tr.ListenAddr)

	assert.Nil(t, tr.ListenAndAccept())
}
