package invoke

import (
	"strings"
	"testing"
)

func TestMethodList(t *testing.T) {
	list := MethodList()
	if len(list) == 0 {
		t.Error("method list is empty")
	}
	for i := 1; i < len(list); i++ {
		if list[i] < list[i-1] {
			t.Errorf("method list not sorted: %s before %s", list[i-1], list[i])
		}
	}
}

func TestFilterMethods(t *testing.T) {
	result := FilterMethods("messages.send")
	if len(result) == 0 {
		t.Error("no methods found for messages.send prefix")
	}
	for _, m := range result {
		if !strings.HasPrefix(m, "messages.send") {
			t.Errorf("unexpected method: %s", m)
		}
	}
}

func TestMethodExists(t *testing.T) {
	if !MethodExists("messages.sendMessage") {
		t.Error("messages.sendMessage should exist")
	}
	if MethodExists("nonexistent.method") {
		t.Error("nonexistent.method should not exist")
	}
}

func TestGetMethodID(t *testing.T) {
	id, err := GetMethodID("messages.sendMessage")
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Error("got zero constructor ID")
	}
}

func TestGetMethodID_NotFound(t *testing.T) {
	_, err := GetMethodID("nonexistent.method")
	if err == nil {
		t.Error("expected error for unknown method")
	}
}
