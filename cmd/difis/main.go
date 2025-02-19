package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Yaroslaw07/difis/pkg/crypto"
	"github.com/Yaroslaw07/difis/pkg/p2p"
	"github.com/Yaroslaw07/difis/pkg/server"
	"github.com/Yaroslaw07/difis/pkg/storage"
	"github.com/Yaroslaw07/difis/pkg/tcp"
)

func makeServer(listenAddr string, nodes ...string) *server.FileServer {
	tcpTransportOpts := tcp.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := tcp.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := server.FileServerOpts{
		EncKey:            crypto.NewEncryptionKey(),
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: storage.CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	fs := server.NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = fs.OnPeer

	return fs
}

func main() {
	fs1 := makeServer(":3000", "")
	fs2 := makeServer(":7000", "")
	fs3 := makeServer(":5000", ":3000", ":7000")

	go func() { log.Fatal(fs1.Start()) }()
	time.Sleep(4 * time.Second)

	go func() { log.Fatal(fs2.Start()) }()
	time.Sleep(4 * time.Second)

	go fs3.Start()
	time.Sleep(2 * time.Second)

	for i := range 4 {
		key := fmt.Sprintf("picture_%d.jpg", i)
		data := bytes.NewReader([]byte("big data file"))
		fs3.Save(key, data)

		if err := fs3.DeleteLocally(key); err != nil {
			log.Fatal(err)
		}

		r, err := fs3.Load(key)
		if err != nil {
			log.Fatal(err)
		}

		b, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(b))
	}
}
