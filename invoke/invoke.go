package invoke

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
)

type Result struct {
	Method   string        `json:"method"`
	Data     interface{}   `json:"data,omitempty"`
	RawBytes []byte        `json:"raw_bytes,omitempty"`
	RawJSON  json.RawMessage `json:"-"`
	Duration time.Duration `json:"duration_ms"`
	Error    string        `json:"error,omitempty"`
}

// InvokeFull calls a TL method by name with JSON params via the full typed path
// using mtgo's JSONClient.InvokeJSON.
func InvokeFull(ctx context.Context, client *telegram.Client, method string, jsonParams []byte) (*Result, error) {
	if !MethodExists(method) {
		return nil, fmt.Errorf("unknown TL method: %s", method)
	}

	start := time.Now()
	jc := telegram.NewJSONClient(client.RPC())
	resp, err := jc.InvokeJSON(ctx, method, jsonParams, false)
	elapsed := time.Since(start)

	result := &Result{
		Method:   method,
		Duration: elapsed,
	}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	result.RawJSON = resp
	return result, nil
}

// InvokeFast calls a TL method via the fast path (skipping TL decode).
// It creates the request struct, unmarshals JSON into it, and uses InvokeWithRawByte.
func InvokeFast(ctx context.Context, client *telegram.Client, method string, jsonParams []byte) (*Result, error) {
	if !MethodExists(method) {
		return nil, fmt.Errorf("unknown TL method: %s", method)
	}

	start := time.Now()

	id := tg.NamesMap[method]
	req, err := findRequestStruct(id)
	if err != nil {
		return nil, fmt.Errorf("create request struct: %w", err)
	}

	if len(jsonParams) > 0 {
		if err := json.Unmarshal(jsonParams, req); err != nil {
			return nil, fmt.Errorf("invalid JSON params: %w", err)
		}
	}

	raw, err := client.InvokeWithRawByte(ctx, req)
	elapsed := time.Since(start)

	result := &Result{
		Method:   method,
		Duration: elapsed,
	}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	result.RawBytes = raw
	return result, nil
}

func findRequestStruct(id uint32) (tg.TLObject, error) {
	if factory, ok := tg.FunctionsMap[id]; ok {
		return factory(), nil
	}
	return nil, fmt.Errorf("no factory for constructor ID 0x%08x", id)
}
