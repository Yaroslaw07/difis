package server

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/Yaroslaw07/difis/pkg/p2p"
)

type MessageType int

const (
	None MessageType = iota
	MessageTypeStore
	MessageTypeGet
	MessageTypeDelete
)

type MessageWrapper struct {
	Payload any
	Type    MessageType
}

type Message struct {
	ID  string
	Key string
}

type MessageGetFile struct {
	Message
}

func newMessageGetFile(id, key string) MessageGetFile {
	return MessageGetFile{
		Message: Message{
			ID:  id,
			Key: key,
		},
	}
}

type MessageStoreFile struct {
	Message
	Size int64
}

const AESBlockSize = 16

func newMessageStoreFile(id, key string, size int64) MessageStoreFile {
	return MessageStoreFile{
		Message: Message{
			ID:  id,
			Key: key,
		},
		Size: size + AESBlockSize,
	}
}

type MessageDeleteFile struct {
	Message
}

func newMessageDeleteFile(id, key string) MessageDeleteFile {
	return MessageDeleteFile{
		Message: Message{
			ID:  id,
			Key: key,
		},
	}
}

func (fs *FileServer) handleMessage(from string, msg *MessageWrapper) error {
	switch v := msg.Type; v {
	case MessageTypeStore:
		if storeMsg, ok := msg.Payload.(MessageStoreFile); ok {
			return fs.handleMessageStoreFile(from, storeMsg)
		}

		return fmt.Errorf("message type store but payload is not of type MessageStoreFile")
	case MessageTypeGet:
		if getMsg, ok := msg.Payload.(MessageGetFile); ok {
			return fs.handleMessageGetFile(from, getMsg)
		}

		return fmt.Errorf("message type get but payload is not of type MessageGetFile")
	case MessageTypeDelete:
		if deleteMsg, ok := msg.Payload.(MessageDeleteFile); ok {
			return fs.handleMessageDeleteFile(from, deleteMsg)
		}
	default:
		return fmt.Errorf("unknown message type: %d", v)
	}

	return nil
}

func (fs *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	n, err := fs.store.Write(msg.ID, msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk\n", fs.Transport.Addr(), n)

	peer.CloseStream()

	return nil
}

func (fs *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !fs.store.Has(msg.ID, msg.Key) {
		return fmt.Errorf("[%s] need to serve but file (%s) doesn't exist on disk", fs.Transport.Addr(), msg.Key)
	}

	fmt.Printf("[%s] got file (%s) that serving over the network\n", fs.Transport.Addr(), msg.Key)

	fileSize, r, err := fs.store.Read(msg.ID, msg.Key)
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

func (fs *FileServer) handleMessageDeleteFile(from string, msg MessageDeleteFile) error {
	return nil
}
