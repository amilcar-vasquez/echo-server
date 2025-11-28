package ws

// Filename: internal/ws/handler.go

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// define request and response message structures
type CommandRequest struct {
	Command string  `json:"command"`
	A       float64 `json:"a"`
	B       float64 `json:"b"`
}

type CommandResponse struct {
	Result  float64 `json:"result,omitempty"`
	Command string  `json:"command"`
	Error   string  `json:"error,omitempty"`
}

// Heartbeat and timeout settings
const (
	writeWait  = 5 * time.Second     // max time to complete a write
	pongWait   = 30 * time.Second    // if we don't get a pong in 30s, time out
	pingPeriod = (pongWait * 9) / 10 // send pings at ~90% of pongWait (e.g., 27s)
)

// Only allow pages served from this origin to connect
var allowedOrigins = []string{
	"http://localhost:4000",
}

func originAllowed(o string) bool {
	if o == "" {
		return false
	}
	for _, a := range allowedOrigins {
		if strings.EqualFold(o, a) {
			return true
		}
	}
	return false
}

func processCommand(payload []byte) ([]byte, error) {
	var req CommandRequest
	// Unmarshal the JSON payload
	err := json.Unmarshal(payload, &req)
	if err != nil {
		// Return error response as JSON
		resp := CommandResponse{
			Command: "unknown",
			Error:   fmt.Sprintf("invalid JSON: %v", err),
		}
		respBytes, _ := json.Marshal(resp)
		return respBytes, nil
	}

	// Switch on req.Command for "add", "subtract", "multiply", "divide"
	var result float64
	var respErr string

	switch req.Command {
	case "add":
		result = req.A + req.B
	case "subtract":
		result = req.A - req.B
	case "multiply":
		result = req.A * req.B
	case "divide":
		if req.B == 0 {
			respErr = "division by zero"
		} else {
			result = req.A / req.B
		}
	default:
		respErr = fmt.Sprintf("unknown command: %s", req.Command)
	}

	// Create command response with result
	resp := CommandResponse{
		Command: req.Command,
	}

	if respErr != "" {
		resp.Error = respErr
	} else {
		resp.Result = result
	}

	// Marshal response to JSON
	respBytes, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	// Return JSON bytes
	return respBytes, nil
}

// The upgrader object is used when we need to upgrade from HTTP to RFC 6455
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		ok := originAllowed(origin)
		if !ok {
			log.Printf("blocked cross-origin websocket: Origin=%q Path=%s", origin, r.URL.Path)
		}
		return ok
	},
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
	},
}

// A simple atomic counter for connection IDs
var messageCounter uint64

// Attempt to upgrade from HTTP to RFC 6455
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Has to be an HTTP GET request
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Upgrade the connection from HTTP to RFC 6455
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("connection opened from %s", r.RemoteAddr)

	// Limit message size
	conn.SetReadLimit(1024 * 4)

	// PING / PONG SETUP

	// Idle timeout window starts now: must receive a pong within pongWait
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))

	// On each pong, extend the read deadline again
	conn.SetPongHandler(func(appData string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		log.Printf("pong from %s (data=%q)", r.RemoteAddr, appData)
		return nil
	})

	// Start a goroutine that sends pings every pingPeriod
	done := make(chan struct{})
	ticker := time.NewTicker(pingPeriod)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Send a ping; if this fails, the read loop will notice soon
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
					log.Printf("ping write error: %v", err)
					return
				}
				log.Printf("ping → %s", r.RemoteAddr)
			case <-done:
				return
			}
		}
	}()

	// Read/Echo loop
	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			// This error will be:
			//  - a timeout (no pong in time), or
			//  - a normal close, or
			//  - some other read error
			log.Printf("read error (timeout/close): %v", err)

			// Try to send a graceful close so the client can see 1000 instead of 1006
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "idle timeout"),
				time.Now().Add(writeWait),
			)

			break
		}

		// We successfully read a message; normal traffic also keeps the connection alive.
		// Note: the pong handler also updates the read deadline on pongs.

		// Echo back text messages, formatting the response to include the message counter
		if msgType == websocket.TextMessage {
			// Increment message counter
			id := atomic.AddUint64(&messageCounter, 1)
			log.Printf("received message #%d from %s: %q", id, r.RemoteAddr, payload)
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))

			message := string(payload)
			var responseBody string

			// Check if the message starts with "UPPER:" or "REVERSE:"
			if strings.HasPrefix(message, "UPPER:") {
				text := strings.TrimPrefix(message, "UPPER:")
				responseBody = strings.ToUpper(text)
			} else if strings.HasPrefix(message, "REVERSE:") {
				text := strings.TrimPrefix(message, "REVERSE:")
				runes := []rune(text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				responseBody = string(runes)
			} else if len(message) > 0 && strings.HasPrefix(message, "{") {
				// Attempt to process as command
				resp, err := processCommand(payload)
				if err != nil {
					responseBody = fmt.Sprintf(`{"error":"%s"}`, err.Error())
				} else {
					responseBody = string(resp)
				}
			} else {
				// Echo back as-is
				responseBody = message
			}

			// Format the response to include the counter
			formatted := "#" + strconv.FormatUint(id, 10) + " " + responseBody

			if err := conn.WriteMessage(websocket.TextMessage, []byte(formatted)); err != nil {
				log.Printf("write error: %v", err)
				break
			}
			log.Printf("echoed message #%d to %s: %q", id, r.RemoteAddr, formatted)
		}
	}

	// Stop the ping goroutine
	close(done)

	log.Printf("connection closed from %s", r.RemoteAddr)
}
