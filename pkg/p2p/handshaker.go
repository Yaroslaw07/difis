package p2p

// Handshake func is
type HandshakeFunc func(Peer) error

func NOPHandshakeFunc(Peer) error { return nil }
