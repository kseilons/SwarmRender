package usecase_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kseilons/job-service/internall/usecase"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestHub_ClientLifecycle(t *testing.T) {
	hub := usecase.NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, _ := upgrader.Upgrade(w, r, nil)
		client := &usecase.Client{
			Hub:    hub,
			Conn:   conn,
			Send:   make(chan []byte, 256),
			UserID: "user-1",
		}
		hub.RegisterClient(client)
		go client.WritePump()
		client.ReadPump()
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]

	t.Run("RegisterAndBroadcast", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		time.Sleep(100 * time.Millisecond) // Wait for registration

		update := map[string]string{"status": "ok"}
		hub.BroadcastStatusUpdate("user-1", update)

		_, message, err := conn.ReadMessage()
		assert.NoError(t, err)

		var received map[string]string
		err = json.Unmarshal(message, &received)
		assert.NoError(t, err)
		assert.Equal(t, "ok", received["status"])
	})

	t.Run("NoBroadcastToOtherUser", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		time.Sleep(100 * time.Millisecond)

		update := map[string]string{"status": "secret"}
		hub.BroadcastStatusUpdate("other-user", update)

		// Set read deadline to ensure it doesn't block forever
		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		_, _, err = conn.ReadMessage()
		assert.Error(t, err) // Should timeout
	})
}
