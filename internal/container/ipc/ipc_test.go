package ipc

import (
	"net"
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
		assert.Equal(t, byte(1), msg)

		msgCh <- msg
	}()

	sendConn, err := socket.Dial()
	assert.NoError(t, err)
	defer sendConn.Close()

	err = SendMessage(sendConn, MsgStart)
	assert.NoError(t, err)

	msg := <-msgCh
	assert.Equal(t, MsgStart, msg)
}

func TestIPCSocketPair(t *testing.T) {
	receiveFile, sendFile, err := NewSocketPair()
	assert.NoError(t, err)

	receiveConn, err := net.FileConn(receiveFile)
	assert.NoError(t, err)
	defer receiveConn.Close()

	sendConn, err := net.FileConn(sendFile)
	assert.NoError(t, err)
	defer sendConn.Close()

	go SendMessage(sendConn, MsgExecReady)
	msg, err := ReceiveMessage(receiveConn)
	assert.NoError(t, err)
	assert.Equal(t, MsgExecReady, msg)
}

func TestShortID(t *testing.T) {
	scenarios := map[string]struct {
		bundle  string
		shortID string
	}{
		"empty bundle": {
			bundle:  "",
			shortID: "e3b0c44298fc1c14",
		},
		"uuid bundle": {
			bundle:  "0f83294c-7d2a-4c63-9694-23cdb7f1c9fb",
			shortID: "87c4f20732570a1a",
		},
		"common bundle": {
			bundle:  "busybox",
			shortID: "9d75f0d7c398df56",
		},
		"absolute path bundle": {
			bundle:  "/tmp/bundledir",
			shortID: "31bf25484a427538",
		},
		"relative path bundle": {
			bundle:  "./bundledir",
			shortID: "4412888c6ed5f1a7",
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			assert.Equal(t, data.shortID, ShortID(data.bundle))
		})
	}
}
