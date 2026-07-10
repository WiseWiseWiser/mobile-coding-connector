## Expected

1. `RegisteredInServer` is true — `server/server.go` mounts/registers the open route.
2. `InAuthSkipList` is false — path is not in the Middleware skip slice.
3. Sources checked path is non-empty.

## Errors

- Route never registered on host mux.
- Adding `/api/local/iterm2/open` to the Middleware skip slice.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.SkipListSource == "" {
		t.Fatal("SkipListSource empty")
	}
	if !resp.RegisteredInServer {
		t.Fatalf("%s must register %s (localiterm2.Register or HandleFunc)", resp.SkipListSource, openEndpoint)
	}
	if resp.InAuthSkipList {
		t.Fatalf("%s must not skip-list %s", resp.SkipListSource, openEndpoint)
	}
}
```

