package p2p

// Peer is an interface that represents node
type Peer interface {
	Close() error
}

// Transport is anything that handles the communication between nodes
// Supports: TCP, UDP, WebSockets, ...
type Transport interface {
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
