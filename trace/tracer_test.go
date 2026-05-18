package trace

import (
	"bytes"
	"testing"
)

func TestTracer(t *testing.T) {
	var buf bytes.Buffer
	tr := NewTracer(&buf)

	id := tr.NextID()
	tr.Tracef(id, ">> test.method")

	output := buf.String()
	if len(output) == 0 {
		t.Error("tracer produced no output")
	}
}

func TestNextID(t *testing.T) {
	tr := NewTracer(nil)
	id1 := tr.NextID()
	id2 := tr.NextID()
	if id2 <= id1 {
		t.Errorf("NextID not incrementing: %d then %d", id1, id2)
	}
}
