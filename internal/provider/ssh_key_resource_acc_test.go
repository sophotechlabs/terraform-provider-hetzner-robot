package provider

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/sophotechlabs/terraform-provider-hetzner-robot/internal/hrobot"
)

func fakeFingerprint(data string) string {
	sum := md5.Sum([]byte(data))
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(parts, ":")
}

func stripKeyComment(data string) string {
	fields := strings.Fields(data)
	if len(fields) >= 2 {
		return fields[0] + " " + fields[1]
	}
	return data
}

func mockKeyServer(t *testing.T, stripComment bool) *httptest.Server {
	t.Helper()

	var mu sync.Mutex
	keys := map[string]hrobot.Key{}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if _, _, ok := r.BasicAuth(); !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"status":401,"code":"UNAUTHORIZED","message":"Unauthorized"}}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = r.ParseForm()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/key":
			data := r.PostForm.Get("data")
			fp := fakeFingerprint(data)
			if _, exists := keys[fp]; exists {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":{"status":409,"code":"KEY_ALREADY_EXISTS","message":"key already exists"}}`))
				return
			}
			stored := data
			if stripComment {
				stored = stripKeyComment(data)
			}
			key := hrobot.Key{
				Name:        r.PostForm.Get("name"),
				Fingerprint: fp,
				Type:        "ED25519",
				Size:        256,
				Data:        stored,
				CreatedAt:   "2026-06-26 10:00:00",
			}
			keys[fp] = key
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]hrobot.Key{"key": key})

		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/key/"):
			fp := strings.TrimPrefix(r.URL.Path, "/key/")
			key, ok := keys[fp]
			if !ok {
				writeKeyNotFound(w)
				return
			}
			key.Name = r.PostForm.Get("name")
			keys[fp] = key
			_ = json.NewEncoder(w).Encode(map[string]hrobot.Key{"key": key})

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/key/"):
			fp := strings.TrimPrefix(r.URL.Path, "/key/")
			key, ok := keys[fp]
			if !ok {
				writeKeyNotFound(w)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]hrobot.Key{"key": key})

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/key/"):
			fp := strings.TrimPrefix(r.URL.Path, "/key/")
			if _, ok := keys[fp]; !ok {
				writeKeyNotFound(w)
				return
			}
			delete(keys, fp)
			w.WriteHeader(http.StatusNoContent)

		default:
			writeKeyNotFound(w)
		}
	}))
}

func writeKeyNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"error":{"status":404,"code":"NOT_FOUND","message":"key not found"}}`))
}

const testSSHKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILab0123456789abcdef k3s-lab"

func TestAccSSHKeyResource_Lifecycle(t *testing.T) {
	server := mockKeyServer(t, false)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactoriesWithServer(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig + fmt.Sprintf(`
resource "hrobot_ssh_key" "test" {
  name       = "lab"
  public_key = %q
}
`, testSSHKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hrobot_ssh_key.test", "name", "lab"),
					resource.TestCheckResourceAttr("hrobot_ssh_key.test", "public_key", testSSHKey),
					resource.TestCheckResourceAttr("hrobot_ssh_key.test", "type", "ED25519"),
					resource.TestCheckResourceAttr("hrobot_ssh_key.test", "size", "256"),
					resource.TestCheckResourceAttrSet("hrobot_ssh_key.test", "id"),
					resource.TestCheckResourceAttrSet("hrobot_ssh_key.test", "fingerprint"),
					resource.TestCheckResourceAttrPair("hrobot_ssh_key.test", "id", "hrobot_ssh_key.test", "fingerprint"),
				),
			},
			{
				Config: testProviderConfig + fmt.Sprintf(`
resource "hrobot_ssh_key" "test" {
  name       = "lab-renamed"
  public_key = %q
}
`, testSSHKey),
				Check: resource.TestCheckResourceAttr("hrobot_ssh_key.test", "name", "lab-renamed"),
			},
			{
				ResourceName:      "hrobot_ssh_key.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSSHKeyResource_ReplaceOnKeyChange(t *testing.T) {
	server := mockKeyServer(t, false)
	defer server.Close()

	const otherKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDifferentKeyMaterial000000000 other"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactoriesWithServer(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig + fmt.Sprintf(`
resource "hrobot_ssh_key" "test" {
  name       = "lab"
  public_key = %q
}
`, testSSHKey),
				Check: resource.TestCheckResourceAttr("hrobot_ssh_key.test", "public_key", testSSHKey),
			},
			{
				Config: testProviderConfig + fmt.Sprintf(`
resource "hrobot_ssh_key" "test" {
  name       = "lab"
  public_key = %q
}
`, otherKey),
				Check: resource.TestCheckResourceAttr("hrobot_ssh_key.test", "public_key", otherKey),
			},
		},
	})
}

func TestAccSSHKeyResource_PreservesConfiguredKeyWhenServerNormalizes(t *testing.T) {
	server := mockKeyServer(t, true)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactoriesWithServer(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig + fmt.Sprintf(`
resource "hrobot_ssh_key" "test" {
  name       = "lab"
  public_key = %q
}
`, testSSHKey),
				Check: resource.TestCheckResourceAttr("hrobot_ssh_key.test", "public_key", testSSHKey),
			},
		},
	})
}
