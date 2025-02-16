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
	fs2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(fs1.Start())
	}()
	time.Sleep(2 * time.Second)

	go fs2.Start()
	time.Sleep(2 * time.Second)

	data := bytes.NewReader([]byte("big data file"))
	fs2.StoreData("cool_picture.jpg", data)
	time.Sleep(5 * time.Millisecond)

	// r, err := fs2.Get("cool_picture.jpg")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// b, err := io.ReadAll(r)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(string(b))
}
