package p2p

// RPC holds data that send
// between two nodes in the network
type RPC struct {
	From    string
	Payload []byte
}
