package p2p

import "net"

// RPC holds data that send
// between two nodes in the network
type RPC struct {
	From    net.Addr
	Payload []byte
}
