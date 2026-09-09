// internal/ws/ws_integration_test.go
package ws

import (
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"io"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httptest.NewServer(NewRouter(logger, NewHub(), nil).Handler())
}

func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil) //nolint:bodyclose // closed through conn.Close()
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestEchoRoundTrip(t *testing.T) {
	srv := testServer(t)
	conn := dial(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws")

	in := map[string]any{"type": "ping", "payload": "hello"}
	raw, _ := json.Marshal(in)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, raw))

	_, resp, err := conn.ReadMessage()
	require.NoError(t, err)

	var got struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(resp, &got))
	assert.Equal(t, "echo", got.Type)
	assert.JSONEq(t, `"hello"`, string(got.Payload))
}

func TestMalformedFrameDoesNotKillSession(t *testing.T) {
	srv := testServer(t)
	conn := dial(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws")

	// Frame 1 : garbage
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("not json{")))
	// Frame 2 : valid message — session must survive
	valid, _ := json.Marshal(map[string]any{"type": "ping", "payload": "alive"})
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, valid))

	expectSequence(t, conn, []string{"error", "echo"})
}

func readTyped(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	_, resp, err := conn.ReadMessage()
	require.NoError(t, err)
	var got struct{ Type string `json:"type"` }
	require.NoError(t, json.Unmarshal(resp, &got))
	return got.Type
}

func expectSequence(t *testing.T, conn *websocket.Conn, want []string) {
	t.Helper()
	for _, wantType := range want {
		gotType := readTyped(t, conn)
		assert.Equal(t, wantType, gotType, "event stream out of order")
	}
}
