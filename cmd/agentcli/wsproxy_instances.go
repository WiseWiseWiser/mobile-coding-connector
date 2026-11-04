package agentcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

// wsProxyInstancesAPI implements the multi-instance subcommands:
//
//	remote-agent ws-proxy list
//	remote-agent ws-proxy add --name NAME --backend self|http://host:port --subdomain SUB
//	remote-agent ws-proxy remove --name NAME
//	remote-agent ws-proxy start --name NAME
//	remote-agent ws-proxy stop --name NAME

type wsProxyInstanceJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	BackendType string `json:"backend_type"`
	BackendURL  string `json:"backend_url"`
	Subdomain   string `json:"subdomain"`
	ListenPort  int    `json:"listen_port"`
	AutoStart   bool   `json:"auto_start"`
	Enabled     bool   `json:"enabled"`
	PublicURL   string `json:"public_url,omitempty"`
	Running     bool   `json:"running"`
	Backend     string `json:"backend"`
}

func wsproxyInstancesGet(c *client.Client) ([]wsProxyInstanceJSON, error) {
	req, err := c.NewRequest("GET", "/api/ws-proxy/instances", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return nil, fmt.Errorf("%s", apiErr.Message)
	}
	var out struct {
		Instances []wsProxyInstanceJSON `json:"instances"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out.Instances, nil
}

func wsproxyInstancesList(getClient func() (*client.Client, error), _ []string) error {
	c, err := getClient()
	if err != nil {
		return err
	}
	instances, err := wsproxyInstancesGet(c)
	if err != nil {
		return err
	}
	if len(instances) == 0 {
		fmt.Println("no ws-proxy instances; add one with: ws-proxy add --name … --backend … --subdomain …")
		return nil
	}
	for _, inst := range instances {
		state := "stopped"
		if inst.Running {
			state = "running"
		}
		fmt.Printf("%-16s %-9s %-28s %s\n", inst.ID, state, strings.TrimPrefix(inst.PublicURL, "https://"), inst.Backend)
	}
	return nil
}

func wsproxyInstancesAdd(getClient func() (*client.Client, error), args []string) error {
	opts := map[string]string{}
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") && i+1 < len(args) {
			opts[strings.TrimPrefix(args[i], "--")] = args[i+1]
			i++
		}
	}
	backend := opts["backend"]
	if backend == "" {
		return fmt.Errorf("--backend is required: self or http://host:port")
	}
	payload := map[string]string{"name": opts["name"], "backend_type": "proxy", "backend_url": backend, "subdomain": opts["subdomain"]}
	if backend == "self" {
		payload["backend_type"] = "self"
		payload["backend_url"] = ""
	}
	body, _ := json.Marshal(payload)

	c, err := getClient()
	if err != nil {
		return err
	}
	req, err := c.NewRequest("POST", "/api/ws-proxy/instances", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add failed: %s", strings.TrimSpace(string(data)))
	}
	fmt.Printf("added instance %s (autostart=%v)\n", opts["subdomain"], opts["autostart"] == "true")
	return nil
}

func wsproxyInstancesRemove(getClient func() (*client.Client, error), args []string) error {
	name := flagValue(args, "name")
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	c, err := getClient()
	if err != nil {
		return err
	}
	req, err := c.NewRequest("DELETE", "/api/ws-proxy/instances/"+name, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remove failed: %s", strings.TrimSpace(string(data)))
	}
	fmt.Println("removed", name)
	return nil
}

func wsproxyInstancesStartStop(getClient func() (*client.Client, error), args []string, action string) error {
	name := flagValue(args, "name")
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	c, err := getClient()
	if err != nil {
		return err
	}
	req, err := c.NewRequest("POST", "/api/ws-proxy/instances/"+name+"/"+action, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s failed: %s", action, strings.TrimSpace(string(data)))
	}
	fmt.Printf("%s: %s done\n", name, action)
	return nil
}

func flagValue(args []string, name string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--"+name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
