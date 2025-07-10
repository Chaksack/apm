package deployment

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

// WebSocketHub manages WebSocket connections for deployment status updates
type WebSocketHub struct {
	clients      map[string]*WebSocketClient
	broadcast    chan StatusUpdate
	register     chan *WebSocketClient
	unregister   chan *WebSocketClient
	deployments  map[string][]string // deployment ID -> client IDs
	mu           sync.RWMutex
}

// WebSocketClient represents a WebSocket client connection
type WebSocketClient struct {
	ID           string
	Connection   *websocket.Conn
	DeploymentID string
	Send         chan StatusUpdate
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:     make(map[string]*WebSocketClient),
		broadcast:   make(chan StatusUpdate, 100),
		register:    make(chan *WebSocketClient),
		unregister:  make(chan *WebSocketClient),
		deployments: make(map[string][]string),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case update := <-h.broadcast:
			h.broadcastUpdate(update)

		case <-ticker.C:
			h.sendPing()
		}
	}
}

// RegisterClient registers a new WebSocket client
func (h *WebSocketHub) RegisterClient(conn *websocket.Conn, deploymentID string) *WebSocketClient {
	client := &WebSocketClient{
		ID:           uuid.New().String(),
		Connection:   conn,
		DeploymentID: deploymentID,
		Send:         make(chan StatusUpdate, 50),
	}

	h.register <- client
	return client
}

// UnregisterClient unregisters a WebSocket client
func (h *WebSocketHub) UnregisterClient(client *WebSocketClient) {
	h.unregister <- client
}

// BroadcastStatusUpdate broadcasts a status update to relevant clients
func (h *WebSocketHub) BroadcastStatusUpdate(update StatusUpdate) {
	h.broadcast <- update
}

// SendToDeployment sends an update to all clients watching a specific deployment
func (h *WebSocketHub) SendToDeployment(deploymentID string, update StatusUpdate) {
	h.mu.RLock()
	clientIDs, exists := h.deployments[deploymentID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	for _, clientID := range clientIDs {
		h.mu.RLock()
		client, exists := h.clients[clientID]
		h.mu.RUnlock()

		if exists {
			select {
			case client.Send <- update:
			default:
				// Client send channel is full, skip
			}
		}
	}
}

// registerClient handles client registration
func (h *WebSocketHub) registerClient(client *WebSocketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client

	// Add client to deployment watchers
	if client.DeploymentID != "" {
		h.deployments[client.DeploymentID] = append(h.deployments[client.DeploymentID], client.ID)
	}

	// Send initial connection success message
	client.Send <- StatusUpdate{
		DeploymentID: client.DeploymentID,
		Timestamp:    time.Now(),
		Type:         "connection",
		Data: map[string]interface{}{
			"status":  "connected",
			"message": "Successfully connected to deployment status stream",
		},
	}
}

// unregisterClient handles client unregistration
func (h *WebSocketHub) unregisterClient(client *WebSocketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.ID]; exists {
		delete(h.clients, client.ID)
		close(client.Send)

		// Remove client from deployment watchers
		if client.DeploymentID != "" {
			clientIDs := h.deployments[client.DeploymentID]
			for i, id := range clientIDs {
				if id == client.ID {
					h.deployments[client.DeploymentID] = append(clientIDs[:i], clientIDs[i+1:]...)
					break
				}
			}
			// Clean up empty deployment entries
			if len(h.deployments[client.DeploymentID]) == 0 {
				delete(h.deployments, client.DeploymentID)
			}
		}
	}
}

// broadcastUpdate broadcasts an update to all relevant clients
func (h *WebSocketHub) broadcastUpdate(update StatusUpdate) {
	if update.DeploymentID != "" {
		// Send to specific deployment watchers
		h.SendToDeployment(update.DeploymentID, update)
	} else {
		// Broadcast to all clients
		h.mu.RLock()
		defer h.mu.RUnlock()

		for _, client := range h.clients {
			select {
			case client.Send <- update:
			default:
				// Client send channel is full, skip
			}
		}
	}
}

// sendPing sends a ping message to all clients
func (h *WebSocketHub) sendPing() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	pingUpdate := StatusUpdate{
		Timestamp: time.Now(),
		Type:      "ping",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	}

	for _, client := range h.clients {
		select {
		case client.Send <- pingUpdate:
		default:
			// Client send channel is full, skip
		}
	}
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(hub *WebSocketHub) func(*websocket.Conn) {
	return func(conn *websocket.Conn) {
		// Get deployment ID from query params
		deploymentID := conn.Query("deployment_id")

		// Register client
		client := hub.RegisterClient(conn, deploymentID)
		defer hub.UnregisterClient(client)

		// Start goroutines for reading and writing
		done := make(chan struct{})
		go client.readPump(done)
		go client.writePump(done)

		// Wait for either goroutine to finish
		<-done
	}
}

// readPump reads messages from the WebSocket connection
func (c *WebSocketClient) readPump(done chan struct{}) {
	defer func() {
		c.Connection.Close()
		close(done)
	}()

	c.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Connection.SetPongHandler(func(string) error {
		c.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		messageType, message, err := c.Connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error
			}
			break
		}

		// Handle client messages (if needed)
		if messageType == websocket.TextMessage {
			// Parse and handle client commands
			var cmd map[string]interface{}
			if err := json.Unmarshal(message, &cmd); err == nil {
				// Handle commands like subscribe/unsubscribe to specific deployments
				if action, ok := cmd["action"].(string); ok {
					switch action {
					case "subscribe":
						if deploymentID, ok := cmd["deployment_id"].(string); ok {
							c.DeploymentID = deploymentID
							// Re-register to update deployment mapping
						}
					case "unsubscribe":
						c.DeploymentID = ""
					}
				}
			}
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *WebSocketClient) writePump(done chan struct{}) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Connection.Close()
		close(done)
	}()

	for {
		select {
		case update, ok := <-c.Send:
			c.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Connection.WriteJSON(update); err != nil {
				return
			}

		case <-ticker.C:
			c.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// StatusStreamer provides methods for streaming deployment status updates
type StatusStreamer struct {
	hub *WebSocketHub
}

// NewStatusStreamer creates a new status streamer
func NewStatusStreamer(hub *WebSocketHub) *StatusStreamer {
	return &StatusStreamer{hub: hub}
}

// StreamDeploymentStatus streams status for a specific deployment
func (s *StatusStreamer) StreamDeploymentStatus(deployment *Deployment) {
	update := StatusUpdate{
		DeploymentID: deployment.ID,
		Timestamp:    time.Now(),
		Type:         UpdateTypeStatus,
		Data: map[string]interface{}{
			"status":      deployment.Status,
			"name":        deployment.Name,
			"version":     deployment.Version,
			"environment": deployment.Environment,
		},
	}
	s.hub.BroadcastStatusUpdate(update)
}

// StreamProgress streams deployment progress
func (s *StatusStreamer) StreamProgress(deploymentID string, progress *DeploymentProgress) {
	update := StatusUpdate{
		DeploymentID: deploymentID,
		Timestamp:    time.Now(),
		Type:         UpdateTypeProgress,
		Data:         progress,
	}
	s.hub.BroadcastStatusUpdate(update)
}

// StreamHealthCheck streams health check results
func (s *StatusStreamer) StreamHealthCheck(deploymentID string, healthChecks []HealthCheck) {
	update := StatusUpdate{
		DeploymentID: deploymentID,
		Timestamp:    time.Now(),
		Type:         UpdateTypeHealth,
		Data:         healthChecks,
	}
	s.hub.BroadcastStatusUpdate(update)
}

// StreamLog streams log messages
func (s *StatusStreamer) StreamLog(deploymentID string, level, message string) {
	update := StatusUpdate{
		DeploymentID: deploymentID,
		Timestamp:    time.Now(),
		Type:         UpdateTypeLog,
		Data: map[string]interface{}{
			"level":   level,
			"message": message,
		},
	}
	s.hub.BroadcastStatusUpdate(update)
}

// StreamError streams error messages
func (s *StatusStreamer) StreamError(deploymentID string, error string) {
	update := StatusUpdate{
		DeploymentID: deploymentID,
		Timestamp:    time.Now(),
		Type:         UpdateTypeError,
		Data: map[string]interface{}{
			"error": error,
		},
	}
	s.hub.BroadcastStatusUpdate(update)
}