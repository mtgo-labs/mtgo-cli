package invoke

import (
	"testing"
)

func TestInvokeFullUnknownMethod(t *testing.T) {
	_, err := InvokeFull(nil, nil, "nonexistent.method", nil)
	if err == nil {
		t.Error("expected error for unknown method")
	}
}

func TestInvokeFastUnknownMethod(t *testing.T) {
	_, err := InvokeFast(nil, nil, "nonexistent.method", nil)
	if err == nil {
		t.Error("expected error for unknown method")
	}
}

func TestFindRequestStruct(t *testing.T) {
	id, _ := GetMethodID("messages.sendMessage")
	req, err := findRequestStruct(id)
	if err != nil {
		t.Fatal(err)
	}
	if req == nil {
		t.Error("request struct is nil")
	}
}
