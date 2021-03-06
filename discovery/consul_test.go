package discovery

import (
	"testing"
	"time"
)

func setupConsul(serviceName string) (Consul, *ServiceDefinition) {
	consul := NewConsulConfig("consul:8500")
	service := &ServiceDefinition{
		ID:        serviceName,
		Name:      serviceName,
		IpAddress: "192.168.1.1",
		TTL:       1,
		Port:      9000,
	}
	return consul, service
}

func TestConsulAddressParse(t *testing.T) {
	// typical valid entries
	runParseTest(t, "https://consul:8500", "consul:8500", "https")
	runParseTest(t, "http://consul:8500", "consul:8500", "http")
	runParseTest(t, "consul:8500", "consul:8500", "http")

	// malformed URI: we won't even try to fix these and just let them bubble up
	// to the Consul API call where it'll fail there.
	runParseTest(t, "httpshttps://consul:8500", "httpshttps://consul:8500", "http")
	runParseTest(t, "https://https://consul:8500", "https://consul:8500", "https")
	runParseTest(t, "http://https://consul:8500", "https://consul:8500", "http")
	runParseTest(t, "consul:8500https://", "consul:8500https://", "http")
	runParseTest(t, "", "", "http")
}

func runParseTest(t *testing.T, uri, expectedAddress, expectedScheme string) {

	address, scheme := parseRawUri(uri)
	if address != expectedAddress || scheme != expectedScheme {
		t.Fatalf("Expected %s over %s but got %s over %s",
			expectedAddress, expectedScheme, address, scheme)
	}
}

func TestConsulTTLPass(t *testing.T) {
	consul, service := setupConsul("service-TestConsulTTLPass")
	id := service.ID

	consul.SendHeartbeat(service) // force registration
	checks, _ := consul.Agent().Checks()
	check := checks[id]
	if check.Status != "critical" {
		t.Fatalf("status of check %s should be 'critical' but is %s", id, check.Status)
	}

	consul.SendHeartbeat(service) // write TTL and verify
	checks, _ = consul.Agent().Checks()
	check = checks[id]
	if check.Status != "passing" {
		t.Fatalf("status of check %s should be 'passing' but is %s", id, check.Status)
	}
}

func TestConsulCheckForChanges(t *testing.T) {
	backend := "service-TestConsulCheckForChanges"
	consul, service := setupConsul(backend)
	id := service.ID
	if consul.CheckForUpstreamChanges(backend, "") {
		t.Fatalf("First read of %s should show `false` for change", id)
	}
	consul.SendHeartbeat(service) // force registration
	consul.SendHeartbeat(service) // write TTL

	if !consul.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should have changed after first health check TTL", id)
	}
	if consul.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should not have changed without TTL expiring", id)
	}
	time.Sleep(2 * time.Second) // wait for TTL to expire
	if !consul.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should have changed after TTL expired.", id)
	}
}
