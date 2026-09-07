package gateway

import (
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

// Command represents an instruction pushed to the Android device.
type Command struct {
	Type        string    `json:"type"`       // "sms.send", "ping"
	MessageID   uuid.UUID `json:"message_id"`
	RequestID   *string   `json:"request_id,omitempty"`
	Recipient   string    `json:"recipient"`
	Content     string    `json:"content"`
	SimSlot     int       `json:"sim_slot"`
	Timestamp   time.Time `json:"timestamp"`
}

// DeviceClient tracks a connected physical Android phone.
type DeviceClient struct {
	PhoneID   uuid.UUID
	Conn      *websocket.Conn
	SendChan  chan Command
	Connected time.Time
}

// Hub orchestrates real-time command delivery to Android devices.
type Hub struct {
	mu           sync.RWMutex
	clients      map[uuid.UUID]*DeviceClient
	pendingQueue map[uuid.UUID][]Command // In-memory fallback queue
	log          *slog.Logger
}

var GlobalHub *Hub
var hubOnce sync.Once

func GetHub(log *slog.Logger) *Hub {
	hubOnce.Do(func() {
		GlobalHub = &Hub{
			clients:      make(map[uuid.UUID]*DeviceClient),
			pendingQueue: make(map[uuid.UUID][]Command),
			log:          log,
		}
	})
	return GlobalHub
}

// Register connects an Android phone socket.
func (h *Hub) Register(phoneID uuid.UUID, conn *websocket.Conn) *DeviceClient {
	h.mu.Lock()
	defer h.mu.Unlock()

	// If prior client exists, close old connection cleanly
	if old, exists := h.clients[phoneID]; exists {
		close(old.SendChan)
		_ = old.Conn.Close()
	}

	client := &DeviceClient{
		PhoneID:   phoneID,
		Conn:      conn,
		SendChan:  make(chan Command, 64),
		Connected: time.Now(),
	}

	h.clients[phoneID] = client
	h.log.Info("Android device connected via WebSocket",
		slog.String("phone_id", phoneID.String()),
		slog.String("remote_addr", conn.RemoteAddr().String()),
	)

	// Flush any pending commands buffered while device was disconnected
	if pending, exists := h.pendingQueue[phoneID]; exists && len(pending) > 0 {
		h.log.Info("flushing pending commands to reconnected Android device",
			slog.String("phone_id", phoneID.String()),
			slog.Int("count", len(pending)),
		)
		for _, cmd := range pending {
			client.SendChan <- cmd
		}
		delete(h.pendingQueue, phoneID)
	}

	return client
}

// Unregister disconnects an Android phone socket.
func (h *Hub) Unregister(phoneID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, exists := h.clients[phoneID]; exists {
		close(client.SendChan)
		_ = client.Conn.Close()
		delete(h.clients, phoneID)
		h.log.Info("Android device disconnected from WebSocket",
			slog.String("phone_id", phoneID.String()),
		)
	}
}

// IsConnected returns whether the device has an active WebSocket.
func (h *Hub) IsConnected(phoneID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.clients[phoneID]
	return exists
}

// DispatchCommand routes an outbound SMS command directly to the Android device.
func (h *Hub) DispatchCommand(phoneID uuid.UUID, cmd Command) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.clients[phoneID]
	if !exists {
		// Device not currently connected via WebSocket: buffer in fallback queue
		h.pendingQueue[phoneID] = append(h.pendingQueue[phoneID], cmd)
		h.log.Info("device offline from WebSocket; buffered command in fallback queue",
			slog.String("phone_id", phoneID.String()),
			slog.String("message_id", cmd.MessageID.String()),
		)
		return nil
	}

	select {
	case client.SendChan <- cmd:
		h.log.Info("outbound SMS command dispatched to Android device",
			slog.String("phone_id", phoneID.String()),
			slog.String("message_id", cmd.MessageID.String()),
			slog.String("recipient", cmd.Recipient),
		)
		return nil
	default:
		// Channel buffer full: buffer in fallback queue
		h.pendingQueue[phoneID] = append(h.pendingQueue[phoneID], cmd)
		return errors.New("device send channel buffer full; command queued")
	}
}

// PollPending retrieves and clears buffered commands for devices polling via HTTP.
func (h *Hub) PollPending(phoneID uuid.UUID) []Command {
	h.mu.Lock()
	defer h.mu.Unlock()

	pending, exists := h.pendingQueue[phoneID]
	if !exists || len(pending) == 0 {
		return []Command{}
	}

	cmds := make([]Command, len(pending))
	copy(cmds, pending)
	delete(h.pendingQueue, phoneID)
	return cmds
}

// WritePump pushes commands from SendChan across the WebSocket connection.
func (c *DeviceClient) WritePump(log *slog.Logger) {
	for cmd := range c.SendChan {
		bytes, err := json.Marshal(cmd)
		if err != nil {
			log.Error("failed to marshal command for Android device", "error", err)
			continue
		}

		_ = c.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.Conn.WriteMessage(websocket.TextMessage, bytes); err != nil {
			log.Warn("error writing to Android device websocket",
				slog.String("phone_id", c.PhoneID.String()),
				slog.String("error", err.Error()),
			)
			return
		}
	}
}
