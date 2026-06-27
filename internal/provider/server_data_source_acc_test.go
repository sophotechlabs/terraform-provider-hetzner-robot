package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/sophotechlabs/terraform-provider-hetzner-robot/internal/hrobot"
)

func mockRobotServer(t *testing.T) *httptest.Server {
	t.Helper()

	servers := map[int]hrobot.Server{
		2893: {
			ServerNumber:  2893,
			ServerIP:      "176.9.18.203",
			ServerIPv6Net: "2a01:4f8:150:5021::",
			ServerName:    "k3s-lab",
			Product:       "AX41",
			DC:            "FSN1-DC8",
			Traffic:       "unlimited",
			Status:        "ready",
			Cancelled:     false,
			PaidUntil:     "2026-12-31",
			IPs:           []string{"176.9.18.203"},
			Subnets:       []hrobot.Subnet{{IP: "2a01:4f8:150:5021::", Mask: "64"}},
		},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"status":401,"code":"UNAUTHORIZED","message":"Unauthorized"}}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/server":
			list := make([]map[string]hrobot.Server, 0, len(servers))
			for _, s := range servers {
				list = append(list, map[string]hrobot.Server{"server": s})
			}
			_ = json.NewEncoder(w).Encode(list)

		case strings.HasPrefix(r.URL.Path, "/server/"):
			num, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/server/"))
			s, ok := servers[num]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":{"status":404,"code":"SERVER_NOT_FOUND","message":"Server not found"}}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]hrobot.Server{"server": s})

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"status":404,"code":"NOT_FOUND","message":"Not found"}}`))
		}
	}))
}

func testProviderFactoriesWithServer(serverURL string) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"hrobot": providerserver.NewProtocol6WithError(
			&HrobotProvider{version: "test", mockEndpoint: serverURL},
		),
	}
}

const testProviderConfig = `
provider "hrobot" {
  username = "test-user"
  password = "test-pass"
}
`

func TestAccServerDataSource_ByNumber(t *testing.T) {
	server := mockRobotServer(t)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactoriesWithServer(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig + `
data "hrobot_server" "test" {
  number = 2893
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.hrobot_server.test", "id", "2893"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "number", "2893"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "name", "k3s-lab"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "product", "AX41"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "datacenter", "FSN1-DC8"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "status", "ready"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "cancelled", "false"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "ipv4", "176.9.18.203"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "ipv6_network", "2a01:4f8:150:5021::"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "ips.#", "1"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "ips.0", "176.9.18.203"),
				),
			},
		},
	})
}

func TestAccServerDataSource_ByIP(t *testing.T) {
	server := mockRobotServer(t)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactoriesWithServer(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig + `
data "hrobot_server" "test" {
  ip = "176.9.18.203"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.hrobot_server.test", "number", "2893"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "id", "2893"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "name", "k3s-lab"),
					resource.TestCheckResourceAttr("data.hrobot_server.test", "ip", "176.9.18.203"),
				),
			},
		},
	})
}

func TestAccServerDataSource_NotFound(t *testing.T) {
	server := mockRobotServer(t)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactoriesWithServer(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig + `
data "hrobot_server" "test" {
  number = 9999
}
`,
				ExpectError: regexp.MustCompile("SERVER_NOT_FOUND"),
			},
		},
	})
}

func TestAccServerDataSource_BothKeysSet(t *testing.T) {
	server := mockRobotServer(t)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactoriesWithServer(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig + `
data "hrobot_server" "test" {
  number = 2893
  ip     = "176.9.18.203"
}
`,
				ExpectError: regexp.MustCompile("exactly one of"),
			},
		},
	})
}

func TestAccServerDataSource_NoKeysSet(t *testing.T) {
	server := mockRobotServer(t)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactoriesWithServer(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig + `
data "hrobot_server" "test" {
}
`,
				ExpectError: regexp.MustCompile("exactly one of"),
			},
		},
	})
}
