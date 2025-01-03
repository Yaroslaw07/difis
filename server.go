package main

import (
	"fmt"
	"io"
	"log"

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

func (fs *FileServer) Store(key string, r io.Reader) error {
	return fs.store.Write(key, r)
}

func (fs *FileServer) loop() {
	defer func() {
		log.Println("File server stopped due to stopped question")
		fs.Transport.Close()
	}()

	for {
		select {
		case msg := <-fs.Transport.Consume():
			fmt.Println(msg)
		case <-fs.quitChannel:
			return
		}
	}
}

func (fs *FileServer) bootstrapNetwork() error {
	for _, addr := range fs.BootStrapNodes {

		if len(addr) == 0 {
			continue
		}

		fmt.Println("attempting to dial ", addr)
		go func(addr string) {
			if err := fs.Transport.Dial(addr); err != nil {
				log.Println("dial error ", err)
			}
		}(addr)
	}

	return nil
}
