package trace

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
)

var idSeq int64

var sensitivePrefixes = []string{
	"auth.",
	"account.updatePassword",
	"account.getPassword",
	"account.resetPassword",
	"account.sendConfirmPhone",
	"messages.requestWebView",
	"messages.prolongWebView",
}

func isSensitive(name string) bool {
	for _, p := range sensitivePrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

type Tracer struct {
	out io.Writer
	mu  sync.Mutex
}

func NewTracer(out io.Writer) *Tracer {
	if out == nil {
		out = os.Stdout
	}
	return &Tracer{out: out}
}

func (t *Tracer) Tracef(id int64, format string, args ...any) {
	t.mu.Lock()
	fmt.Fprintf(t.out, "[%d] %s\n", id, fmt.Sprintf(format, args...))
	t.mu.Unlock()
}

func (t *Tracer) NextID() int64 {
	return atomic.AddInt64(&idSeq, 1)
}

func (t *Tracer) Middleware() telegram.InvokerMiddleware {
	return func(next tg.Invoker) tg.Invoker {
		return tg.InvokerFunc(func(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
			id := t.NextID()
			name := tgName(input)

			if isSensitive(name) {
				t.Tracef(id, ">> %s [REDACTED]", name)
			} else {
				t.Tracef(id, ">> %s", name)
				t.Tracef(id, "   %v", input)
			}

			start := time.Now()
			result, err := next.RPCInvoke(ctx, input, decode)
			elapsed := time.Since(start)

			if err != nil {
				t.Tracef(id, "<< %s [ERROR] [%s]", name, elapsed.Round(time.Millisecond))
			} else {
				t.Tracef(id, "<< %s [%s]", name, elapsed.Round(time.Millisecond))
				if !isSensitive(name) {
					t.Tracef(id, "   %v", result)
				}
			}
			return result, err
		})
	}
}

func (t *Tracer) UpdateHandler() func(ctx *telegram.Context) {
	return func(ctx *telegram.Context) {
		id := t.NextID()
		name := tgName(ctx.Update.Raw)
		t.Tracef(id, "UPDATE %s", name)
	}
}

func tgName(obj tg.TLObject) string {
	return fmt.Sprintf("%T", obj)
}
