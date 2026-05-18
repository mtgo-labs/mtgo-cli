package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

type Request struct {
	Method  string      `json:"method"`
	Payload interface{} `json:"payload,omitempty"`
}

type Response struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
	DurMs int64       `json:"duration_ms,omitempty"`
}

type InvokePayload struct {
	TLMethod   string          `json:"tl_method"`
	JSONParams json.RawMessage `json:"json_params"`
	Fast       bool            `json:"fast"`
}

type Server struct {
	socketPath string
	listener   net.Listener
	handler    Handler
	mu         sync.Mutex
	running    bool
}

type Handler interface {
	HandleInvoke(payload InvokePayload) (*Response, error)
	HandleStatus() *Response
}

func NewServer(socketPath string, handler Handler) *Server {
	return &Server{
		socketPath: socketPath,
		handler:    handler,
	}
}

func (s *Server) Start() error {
	os.Remove(s.socketPath)

	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("ipc listen: %w", err)
	}
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		l.Close()
		return fmt.Errorf("ipc chmod: %w", err)
	}

	s.listener = l
	s.running = true

	go s.acceptLoop()
	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
		os.Remove(s.socketPath)
	}
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var req Request
	if err := decoder.Decode(&req); err != nil {
		encoder.Encode(Response{OK: false, Error: fmt.Sprintf("invalid request: %v", err)})
		return
	}

	var resp *Response
	var err error

	switch req.Method {
	case "invoke":
		var p InvokePayload
		if data, _ := json.Marshal(req.Payload); len(data) > 0 {
			json.Unmarshal(data, &p)
		}
		resp, err = s.handler.HandleInvoke(p)
	case "status":
		resp = s.handler.HandleStatus()
	case "shutdown":
		resp = &Response{OK: true}
		encoder.Encode(resp)
		go s.Stop()
		return
	default:
		resp = &Response{OK: false, Error: fmt.Sprintf("unknown method: %s", req.Method)}
	}

	if err != nil {
		resp = &Response{OK: false, Error: err.Error()}
	}
	encoder.Encode(resp)
}

func IsSocketActive(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
