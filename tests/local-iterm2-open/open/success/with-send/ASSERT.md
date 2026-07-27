## Expected

1. Status 200 with `ok:true`.
2. `RecordedSend` equals `["echo hi", "ls"]` in order.

```go
import (
	"reflect"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d body=%s", resp.StatusCode, resp.Body)
	}
	if !resp.OK {
		t.Fatalf("want ok:true; body=%s", resp.Body)
	}
	if !resp.OpenCalled {
		t.Fatal("Open not called")
	}
	want := []string{"echo hi", "ls"}
	if !reflect.DeepEqual(resp.RecordedSend, want) {
		t.Fatalf("RecordedSend = %#v, want %#v", resp.RecordedSend, want)
	}
}
```
