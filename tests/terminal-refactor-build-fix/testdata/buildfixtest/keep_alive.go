package buildfixtest

import (
	"bytes"

	"github.com/xhd2015/ai-critic/run"
)

func outputKeepAliveScriptForTest(port int, serverArgs []string, binPath string) (string, error) {
	var buf bytes.Buffer
	if err := run.TestExported_OutputKeepAliveScript(port, serverArgs, binPath, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}