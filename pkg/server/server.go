package server

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/Yaroslaw07/difis/pkg/crypto"
	"github.com/Yaroslaw07/difis/pkg/p2p"
	"github.com/Yaroslaw07/difis/pkg/storage"
)

type FileServerOpts struct {
	ID                string
	EncKey            []byte
	StorageRoot       string
	PathTransformFunc storage.PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store       *storage.Store
	quitChannel chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := storage.StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	if len(opts.ID) == 0 {
		opts.ID = crypto.GenerateID()
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          storage.NewStore(storeOpts),
		quitChannel:    make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
	gob.Register(MessageDeleteFile{})
	gob.Register(MessageWrapper{})
}

func (fs *FileServer) Start() error {
	fmt.Printf("[%s] starting file server...", fs.Transport.Addr())
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

func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.Has(fs.ID, key) {
		fmt.Printf("[%s] serving file (%s) from local disk\n", fs.Transport.Addr(), key)

		_, r, err := fs.store.Read(fs.ID, key)
		return r, err
	}

	fmt.Printf("[%s] don't have file (%s) locally, fetching from network...\n", fs.Transport.Addr(), key)

	msg := MessageWrapper{
		Type:    MessageTypeGet,
		Payload: newMessageGetFile(fs.ID, crypto.HashKey(key)),
	}

	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)

	for _, peer := range fs.peers {
		// First read the file size to limit reader
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)

		n, err := fs.store.WriteDecrypt(fs.EncKey, fs.ID, key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over the network from (%s)", fs.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()
	}

	_, r, err := fs.store.Read(fs.ID, key)

	return r, err
}

func (fs *FileServer) StoreData(key string, r io.Reader) error {
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := fs.store.Write(fs.ID, key, tee)

	if err != nil {
		return err
	}

	msg := MessageWrapper{
		Type:    MessageTypeStore,
		Payload: newMessageStoreFile(fs.ID, crypto.HashKey(key), size),
	}

	if err := fs.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 100)

	peers := []io.Writer{}
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})
	n, err := crypto.CopyEncrypt(fs.EncKey, fileBuffer, mw)

	if err != nil {
		return err
	}

	fmt.Printf("[%s] received and written (%v) bytes to disk\n", fs.Transport.Addr(), n)

	return nil
}

func (fs *FileServer) Store(key string, r io.Reader) error {
	_, err := fs.store.Write(fs.ID, key, r)
	return err
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
		case rpc := <-fs.Transport.Consume():
			var msg MessageWrapper

			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error: ", err)
			}

			if err := fs.handleMessage(rpc.From, &msg); err != nil {
				log.Println("handling message error: ", err)
			}

		case <-fs.quitChannel:
			return
		}
	}
}

func (fs *FileServer) broadcast(msg *MessageWrapper) error {
	buf := new(bytes.Buffer)

	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range fs.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileServer) bootstrapNetwork() error {
	for _, addr := range fs.BootstrapNodes {

		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			fmt.Printf("[%s] attempting to connect with remote %s\n", fs.Transport.Addr(), addr)
			if err := fs.Transport.Dial(addr); err != nil {
				log.Println("dial error ", err)
			}
		}(addr)
	}

	return nil
}
