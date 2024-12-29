package main

import (
	"log"

	"github.com/Yaroslaw07/difis/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// TODO: implement OnPeer
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootStrapNodes:    nodes,
	}

	return NewFileServer(fileServerOpts)
}

func main() {

	fs1 := makeServer(":3000", "")
	fs2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(fs1.Start())
	}()

	fs2.Start()
}
