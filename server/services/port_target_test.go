package services

import (
	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/ai-critic/server/proxy/portforward"
	"testing"
)

func TestServiceForwardDoesNotReplaceRemoteTargetOnSamePort(t *testing.T) {
	pfm := portforward.NewManager()
	provider := &testPortForwardProvider{name: portforward.ProviderCloudflareOwned}
	pfm.RegisterProvider(provider)
	target := config.OriginTarget{Scheme: "http", Host: "192.168.1.1", Port: 9476}
	remote, err := pfm.AddForward(portforward.ForwardRequest{Target: &target, Label: "remote.example", Provider: provider.name})
	if err != nil {
		t.Fatal(err)
	}
	defer pfm.RemoveSelector(remote.ID)
	m := &Manager{processes: map[string]*serviceProcess{"svc-local": {}}, portForwardManager: pfm}
	def := ServiceDefinition{PortForward: &ServicePortForward{Port: 9476, Label: "local.example", Provider: provider.name}}
	if err := m.ensurePortForward("svc-local", def); err != nil {
		t.Fatal(err)
	}
	defer pfm.RemoveSelector(config.LocalMappingID(9476))
	if len(pfm.List()) != 2 {
		t.Fatal("service removed remote same-port target")
	}
	if err := pfm.RemoveSelector(config.LocalMappingID(9476)); err != nil {
		t.Fatal(err)
	}
	left := pfm.List()
	if len(left) != 1 || left[0].ID != remote.ID {
		t.Fatal("stopping local service removed the remote mapping")
	}
}
