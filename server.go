package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/Yaroslaw07/difis/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootStrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store       *Store
	quitChannel chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitChannel:    make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

func (fs *FileServer) Start() error {
	if err := fs.Transport.ListenAndAccept(); err != nil {
		return err
	}

	fs.bootstrapNetwork()

	fs.loop()

	return nil
}

func (fs *FileServer) Stop() {
	close(fs.quitChannel)
}

type Message struct {
	From    string
	Payload any
}

type DataMessage struct {
	Key  string
	Data []byte
}

func (fs *FileServer) StoreData(key string, r io.Reader) error {
	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)

	if err := fs.store.Write(key, tee); err != nil {
		return err
	}

	p := &DataMessage{
		Key:  key,
		Data: buf.Bytes(),
	}

	fmt.Println(p.Data)

	return fs.broadcast(&Message{
		From:    "TODO",
		Payload: p,
	})
}

func (fs *FileServer) Store(key string, r io.Reader) error {
	return fs.store.Write(key, r)
}

func (fs *FileServer) OnPeer(p p2p.Peer) error {
	fs.peerLock.Lock()
	defer fs.peerLock.Unlock()

	fs.peers[p.RemoteAddr().String()] = p

	return nil
}

func (fs *FileServer) loop() {
	defer func() {
		log.Println("File server stopped due to stopped question")
		fs.Transport.Close()
	}()

	for {
		select {
		case msg := <-fs.Transport.Consume():
			var m Message

			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&m); err != nil {
				log.Println(err)
			}

			if err := fs.handleMessage(&m); err != nil {
				log.Println(err)
			}
		case <-fs.quitChannel:
			return
		}
	}
}

func (fs *FileServer) handleMessage(msg *Message) error {
	switch v := msg.Payload.(type) {
	case *DataMessage:
		fmt.Printf("received data %+v\n", v)
	}

	return nil
}

func (fs *FileServer) broadcast(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (fs *FileServer) bootstrapNetwork() error {
	for _, addr := range fs.BootStrapNodes {

		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			if err := fs.Transport.Dial(addr); err != nil {
				log.Println("dial error ", err)
			}
		}(addr)
	}

	return nil
}
