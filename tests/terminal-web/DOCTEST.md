# Durable web terminals

Real PTYs through an in-process HTTP/WebSocket server (L2). The principal
branch is running vs exited vs missing; disconnect variants refine running.
CLI defaults are covered separately as a compatibility boundary. Backend
restart persistence and real iOS suspension are outside this suite.

## DSN (Domain Specific Notion)

Participants: isolated PTY manager, foreground child, WebSocket client, and REST session API.
Behaviors: durable disconnect/replay/input, explicit exit/delete, and legacy writer compatibility.

## Version

0.0.1

## Decision Tree

```text
terminal-web/
├── running/{abrupt,normal,overlap,delete}
├── exited
├── missing
└── legacy
```

## How to Run

```sh
doctest vet ./tests/terminal-web
doctest test ./tests/terminal-web
```

```go
import (
    "testing"
    "github.com/xhd2015/doctest/session"
    "github.com/xhd2015/ai-critic/tests/terminal-web/testdata/webtest"
)
type Request = webtest.Request
type Response = webtest.Response
func Run(t *testing.T, _ *session.Doctest, req *Request) (*Response, error) {
    return webtest.Run(t, req)
}
```
