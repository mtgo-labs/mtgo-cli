package ipc

import (
	"path/filepath"
	"testing"
	"time"
)

type testHandler struct{}

func (h *testHandler) HandleInvoke(payload InvokePayload) (*Response, error) {
	return &Response{OK: true, Data: map[string]string{"echo": string(payload.JSONParams)}, DurMs: 1}, nil
}

func (h *testHandler) HandleStatus() *Response {
	return &Response{OK: true, Data: map[string]bool{"connected": true}}
}

func TestIPCRoundTrip(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "test.sock")
	srv := NewServer(socket, &testHandler{})

	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	time.Sleep(10 * time.Millisecond)

	client := NewClient(socket)

	resp, err := client.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Error("status failed")
	}

	invokeResp, err := client.Invoke(InvokePayload{
		TLMethod:   "test.method",
		JSONParams: []byte(`{"key":"value"}`),
		Fast:       false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !invokeResp.OK {
		t.Error("invoke failed")
	}
}

func TestSocketActivity(t *testing.T) {
	if IsSocketActive("/nonexistent/path.sock") {
		t.Error("expected false for nonexistent socket")
	}
}
