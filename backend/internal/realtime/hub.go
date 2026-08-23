package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type Event struct {
	Type        string `json:"type"`
	ScreeningID string `json:"screeningId"`
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan []byte]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string]map[chan []byte]struct{})}
}

func (h *Hub) Publish(screeningID string) {
	payload, _ := json.Marshal(Event{Type: "seats.updated", ScreeningID: screeningID})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for subscriber := range h.subscribers[screeningID] {
		select {
		case subscriber <- payload:
		default:
		}
	}
}

func (h *Hub) Serve(w http.ResponseWriter, r *http.Request, screeningID, webOrigin string) {
	options := &websocket.AcceptOptions{}
	if origin, err := url.Parse(webOrigin); err == nil && origin.Host != "" {
		options.OriginPatterns = []string{origin.Host}
	}
	connection, err := websocket.Accept(w, r, options)
	if err != nil {
		return
	}
	defer connection.CloseNow()

	messages := make(chan []byte, 8)
	h.subscribe(screeningID, messages)
	defer h.unsubscribe(screeningID, messages)

	ctx := connection.CloseRead(r.Context())
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-messages:
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := connection.Write(writeCtx, websocket.MessageText, message)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (h *Hub) subscribe(screeningID string, messages chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subscribers[screeningID] == nil {
		h.subscribers[screeningID] = make(map[chan []byte]struct{})
	}
	h.subscribers[screeningID][messages] = struct{}{}
}

func (h *Hub) unsubscribe(screeningID string, messages chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subscribers[screeningID], messages)
	if len(h.subscribers[screeningID]) == 0 {
		delete(h.subscribers, screeningID)
	}
}
