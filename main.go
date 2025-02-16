package main

import (
	"bytes"
	"fmt"
	"io"
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
		EncKey:            newEncryptionKey(),
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	fs := NewFileServer(fileServerOpts)

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
		fs3.StoreData(key, data)

		if err := fs3.store.Delete(fs3.ID, key); err != nil {
			log.Fatal(err)
		}

		r, err := fs3.Get(key)
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
