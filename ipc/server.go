package ipc

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	maxConns      = 64
	maxRequestSize = 10 << 20 // 10 MB
	readDeadline  = 30 * time.Second
)

type Request struct {
	Method  string          `json:"method"`
	Payload json.RawMessage `json:"payload,omitempty"`
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
	stopOnce   sync.Once
	wg         sync.WaitGroup
	sem        chan struct{}
}

type Handler interface {
	HandleInvoke(payload InvokePayload) (*Response, error)
	HandleStatus() *Response
}

func NewServer(socketPath string, handler Handler) *Server {
	return &Server{
		socketPath: socketPath,
		handler:    handler,
		sem:        make(chan struct{}, maxConns),
	}
}

func (s *Server) Start() error {
	if fi, err := os.Lstat(s.socketPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("ipc: socket path %s is a symlink, refusing", s.socketPath)
		}
		if fi.Mode()&os.ModeSocket != 0 {
			os.Remove(s.socketPath)
		}
	}

	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("ipc listen: %w", err)
	}

	fi, err := os.Lstat(s.socketPath)
	if err != nil || fi.Mode()&os.ModeSymlink != 0 {
		l.Close()
		return fmt.Errorf("ipc: socket path changed to symlink after creation, refusing")
	}

	if err := os.Chmod(s.socketPath, 0600); err != nil {
		l.Close()
		return fmt.Errorf("ipc chmod: %w", err)
	}

	s.listener = l

	s.wg.Add(1)
	go s.acceptLoop()
	return nil
}

func (s *Server) Stop() error {
	s.stopOnce.Do(func() {
		if s.listener != nil {
			s.listener.Close()
		}
	})
	s.wg.Wait()
	os.Remove(s.socketPath)
	return nil
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}

		if !s.checkPeerCred(conn) {
			conn.Close()
			continue
		}

		s.sem <- struct{}{}
		s.wg.Add(1)
		go func() {
			defer func() {
				<-s.sem
				s.wg.Done()
			}()
			s.handleConn(conn)
		}()
	}
}

func (s *Server) checkPeerCred(conn net.Conn) bool {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return true
	}

	rawConn, err := uc.SyscallConn()
	if err != nil {
		return true
	}

	var peerUID uint32
	ctrlErr := rawConn.Control(func(fd uintptr) {
		ucred, err := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if err != nil {
			return
		}
		peerUID = ucred.Uid
	})
	if ctrlErr != nil {
		return true
	}

	return peerUID == uint32(os.Getuid())
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		return
	}

	limited := io.LimitReader(conn, maxRequestSize)
	decoder := json.NewDecoder(limited)
	encoder := json.NewEncoder(conn)

	var req Request
	if err := decoder.Decode(&req); err != nil {
		encoder.Encode(Response{OK: false, Error: "bad request"})
		return
	}

	var resp *Response
	var err error

	switch req.Method {
	case "invoke":
		var p InvokePayload
		if len(req.Payload) > 0 {
			if err := json.Unmarshal(req.Payload, &p); err != nil {
				encoder.Encode(Response{OK: false, Error: "invalid invoke payload"})
				return
			}
		}
		resp, err = s.handler.HandleInvoke(p)
	case "status":
		resp = s.handler.HandleStatus()
	default:
		resp = &Response{OK: false, Error: "unknown method"}
	}

	if err != nil {
		resp = &Response{OK: false, Error: "internal error"}
	}
	encoder.Encode(resp)
}

func IsSocketActive(socketPath string) bool {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
