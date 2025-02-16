package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/Yaroslaw07/difis/p2p"
)

type FileServerOpts struct {
	EncKey            []byte
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
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

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
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
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.Has(key) {
		fmt.Printf("[%s] serving file (%s) from local disk\n", fs.Transport.Addr(), key)

		_, r, err := fs.store.Read(key)
		return r, err
	}

	fmt.Printf("[%s] don't have file (%s) locally, fetching from network...\n", fs.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)

	for _, peer := range fs.peers {
		// First read the file size to limit reader
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)

		n, err := fs.store.WriteDecrypt(fs.EncKey, key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over the network from (%s)", fs.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()
	}

	_, r, err := fs.store.Read(key)

	return r, err
}

func (fs *FileServer) StoreData(key string, r io.Reader) error {

	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := fs.store.Write(key, tee)

	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size + 16,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 5)

	// TODO: use a multiwriter
	for _, peer := range fs.peers {
		peer.Send([]byte{p2p.IncomingStream})
		n, err := copyEncrypt(fs.EncKey, fileBuffer, peer)
		if err != nil {
			return nil
		}

		fmt.Printf("received and written (%v) bytes\n", n)
	}

	return nil
}

func (fs *FileServer) Store(key string, r io.Reader) error {
	_, err := fs.store.Write(key, r)
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
			var msg Message

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

func (fs *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return fs.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return fs.handleMessageGetFile(from, v)
	}

	return nil
}

func (fs *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := fs.peers[from]

	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	n, err := fs.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk\n", fs.Transport.Addr(), n)
	peer.CloseStream()

	return nil
}

func (fs *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !fs.store.Has(msg.Key) {
		return fmt.Errorf("[%s] need to serve but file (%s) doesn't exist on disk", fs.Transport.Addr(), msg.Key)
	}

	fmt.Printf("[%s] got file (%s) that serving over the network\n", fs.Transport.Addr(), msg.Key)

	fileSize, r, err := fs.store.Read(msg.Key)
	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok {
		fmt.Println("closing read closer")
		defer rc.Close()
	}

	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}

	// First send the "incomingStream" byte to the peer
	// Then we can send the file size (int64)
	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fileSize)

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written (%d) bytes over the network to %s\n", fs.Transport.Addr(), n, from)

	return nil
}

func (fs *FileServer) broadcast(msg *Message) error {
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
			if err := fs.Transport.Dial(addr); err != nil {
				log.Println("dial error ", err)
			}
		}(addr)
	}

	return nil
}
