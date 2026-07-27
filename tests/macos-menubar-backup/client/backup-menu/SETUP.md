# Scenario

**Feature**: Backup menu present in remote menubar app

```
remote AICriticApp.swift -> Menu("Backup") / backup accessibility id
```

## Preconditions

Remote app dropdown includes a Backup entry alongside Server/Services/Terminals.

## Steps

1. Set `ClientLeaf=backup-menu`.

## Context

REQUIREMENT #24. Also expects Backup Now… and Reveal in Finder… as part of structure.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "backup-menu"
	return nil
}
```
