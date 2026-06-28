## Expected

- Route rules include `192.168.0.0/16` (and 10.0.0.0/8, 172.16.0.0/12) with outbound `direct`.

## Side Effects

- None.

## Errors

- None.

## Exit Code

- Success.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("build error: %v", resp.RunErr)
	}
	rules := routeRules(resp.ConfigJSON)
	if len(rules) == 0 {
		t.Fatal("route rules missing")
	}
	found192 := false
	found10 := false
	found172 := false
	for _, r := range rules {
		cidrs, _ := r["ip_cidr"].([]any)
		out, _ := r["outbound"].(string)
		for _, c := range cidrs {
			s, _ := c.(string)
			switch s {
			case "192.168.0.0/16":
				if out == "direct" {
					found192 = true
				}
			case "10.0.0.0/8":
				if out == "direct" {
					found10 = true
				}
			case "172.16.0.0/12":
				if out == "direct" {
					found172 = true
				}
			}
		}
	}
	if !found192 {
		t.Fatal("missing 192.168.0.0/16 -> direct route rule")
	}
	if !found10 {
		t.Fatal("missing 10.0.0.0/8 -> direct route rule")
	}
	if !found172 {
		t.Fatal("missing 172.16.0.0/12 -> direct route rule")
	}
}
```