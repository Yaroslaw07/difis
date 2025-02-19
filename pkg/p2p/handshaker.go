package p2p

// Handshake function is used to perform a handshake between two peers
type HandshakeFunc func(Peer) error

func NOPHandshakeFunc(Peer) error { return nil }
