package main

import (
	"bytes"
	"log"
	"time"

	"github.com/Yaroslaw07/difis/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootStrapNodes:    nodes,
	}

	fs := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = fs.OnPeer

	return fs
}

func main() {
	fs1 := makeServer(":3000", "")
	fs2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(fs1.Start())
	}()
	time.Sleep(1 * time.Second)

	go fs2.Start()
	time.Sleep(1 * time.Second)

	data := bytes.NewReader([]byte("big data file"))
	fs2.StoreData("private_data", data)

	select {}
}
