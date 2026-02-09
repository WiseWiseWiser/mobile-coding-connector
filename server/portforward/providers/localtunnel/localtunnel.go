package localtunnel

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/portforward"
)

// Provider implements portforward.Provider using npx localtunnel
type Provider struct{}

var _ portforward.Provider = (*Provider)(nil)

func (p *Provider) Name() string        { return portforward.ProviderLocaltunnel }
func (p *Provider) DisplayName() string { return "localtunnel" }
func (p *Provider) Description() string {
	return "Free tunneling via loca.lt (npx localtunnel). No account required."
}
func (p *Provider) Available() bool { return portforward.IsCommandAvailable("npx") }

func (p *Provider) Start(port int, _ string) (*portforward.TunnelHandle, error) {
	logs := portforward.NewLogBuffer()

	cmd := exec.Command("npx", "--yes", "localtunnel", "--port", fmt.Sprintf("%d", port))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %v", err)
	}
	cmd.Stderr = logs

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start localtunnel: %v", err)
	}

	resultCh := make(chan portforward.TunnelResult, 1)

	// localtunnel prints: "your url is: https://xxx.loca.lt"
	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.loca\.lt`)

	go func() {
		scanner := bufio.NewScanner(stdout)
		urlFound := make(chan string, 1)

		go func() {
			for scanner.Scan() {
				line := scanner.Text()
				logs.Write([]byte(line + "\n"))
				if match := urlRegex.FindString(line); match != "" {
					urlFound <- match
					return
				}
			}
		}()

		select {
		case url := <-urlFound:
			resultCh <- portforward.TunnelResult{PublicURL: url}
		case <-time.After(60 * time.Second):
			resultCh <- portforward.TunnelResult{Err: fmt.Errorf("timeout waiting for localtunnel URL (60s)")}
			cmd.Process.Kill()
			return
		}

		cmd.Wait()
	}()

	return &portforward.TunnelHandle{
		Result: resultCh,
		Logs:   logs,
		Stop: func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		},
	}, nil
}
