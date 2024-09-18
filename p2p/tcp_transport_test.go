package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	opts := TCPTransportOps{
		ListenAddr:    ":3000",
		Decoder:       DefaultDecoder{},
		HandshakeFunc: NOPHandshakeFunc,
	}
	tr := NewTCPTransport(opts)

	assert.Equal(t, ":3000", tr.ListenAddr)

	assert.Nil(t, tr.ListenAndAccept())
}
