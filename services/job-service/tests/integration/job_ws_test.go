//go:build integration

package integration

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *JobIntegrationTestSuite) Test_WebSocket_Auth() {
	t := s.T()
	wsURL := "ws://localhost:8081/ws"
	token := s.createToken("test-user-ws")

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	authMsg := map[string]string{
		"type":  "auth",
		"token": token,
	}
	err = conn.WriteJSON(authMsg)
	assert.NoError(t, err)

	err = conn.SetReadDeadline(time.Now().Add(time.Second))
	assert.NoError(t, err)
	_, _, err = conn.ReadMessage()
	assert.True(t, websocket.IsUnexpectedCloseError(err) || err != nil)
}
