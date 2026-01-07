package ipc

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPCSocket(t *testing.T) {
	socketDir := t.TempDir()
	socketPath := filepath.Join(socketDir, "test.sock")

	socket := NewSocket(socketPath)

	listener, err := socket.Listen()
	assert.NoError(t, err)
	defer listener.Close()

	msgCh := make(chan byte, 1)
	go func() {
		receiveConn, err := listener.Accept()
		assert.NoError(t, err)
		defer receiveConn.Close()

		msg, err := ReceiveMessage(receiveConn)
		assert.NoError(t, err)
		assert.Equal(t, "testing", msg)

		msgCh <- msg
	}()

	sendConn, err := socket.Dial()
	assert.NoError(t, err)
	defer sendConn.Close()

	err = SendMessage(sendConn, byte(1))
	assert.NoError(t, err)

	msg := <-msgCh
	assert.Equal(t, "testing", msg)
}

func TestIPCSocketPair(t *testing.T) {
	receiveFD, sendFD, err := NewSocketPair()
	assert.NoError(t, err)

	receiveConn, err := FDToConn(receiveFD)
	assert.NoError(t, err)
	defer receiveConn.Close()

	sendConn, err := FDToConn(sendFD)
	assert.NoError(t, err)
	defer sendConn.Close()

	go SendMessage(sendConn, byte(1))
	msg, err := ReceiveMessage(receiveConn)
	assert.NoError(t, err)
	assert.Equal(t, "testing", msg)
}
