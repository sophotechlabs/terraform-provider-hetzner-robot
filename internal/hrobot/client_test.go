package hrobot

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewClient_SetsFields(t *testing.T) {
	c := NewClient("https://robot-ws.your-server.de", "user", "secret")

	if c.Username != "user" {
		t.Errorf("Username = %q, want %q", c.Username, "user")
	}
	if c.Password != "secret" {
		t.Errorf("Password = %q, want %q", c.Password, "secret")
	}
	if c.Endpoint != "https://robot-ws.your-server.de" {
		t.Errorf("Endpoint = %q, want %q", c.Endpoint, "https://robot-ws.your-server.de")
	}
	if c.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestNewClient_DefaultsEndpoint(t *testing.T) {
	c := NewClient("", "user", "secret")
	if c.Endpoint != DefaultEndpoint {
		t.Errorf("Endpoint = %q, want default %q", c.Endpoint, DefaultEndpoint)
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("https://example.com/", "user", "secret")
	if c.Endpoint != "https://example.com" {
		t.Errorf("Endpoint = %q, want %q", c.Endpoint, "https://example.com")
	}
}

func TestAPIError_Error(t *testing.T) {
	withCode := &APIError{Status: 404, Code: "SERVER_NOT_FOUND", Message: "Server not found"}
	if !strings.Contains(withCode.Error(), "SERVER_NOT_FOUND") || !strings.Contains(withCode.Error(), "404") {
		t.Errorf("Error() = %q, want code and status", withCode.Error())
	}

	noCode := &APIError{Status: 500, Message: "boom"}
	if strings.Contains(noCode.Error(), "::") || !strings.Contains(noCode.Error(), "boom") {
		t.Errorf("Error() = %q, want message", noCode.Error())
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(&APIError{Status: http.StatusNotFound}) {
		t.Error("IsNotFound should be true for status 404")
	}
	if IsNotFound(&APIError{Status: http.StatusInternalServerError}) {
		t.Error("IsNotFound should be false for status 500")
	}
	if IsNotFound(io.EOF) {
		t.Error("IsNotFound should be false for a non-APIError")
	}
	if IsNotFound(nil) {
		t.Error("IsNotFound should be false for nil")
	}
}

func TestDo_GetSetsBasicAuthAndAccept(t *testing.T) {
	var gotUser, gotPass, gotAccept string
	var gotAuthOK bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, gotAuthOK = r.BasicAuth()
		gotAccept = r.Header.Get("Accept")
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, Username: "user", Password: "secret", HTTPClient: server.Client()}
	if _, err := c.do(context.Background(), http.MethodGet, "/server", nil); err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !gotAuthOK || gotUser != "user" || gotPass != "secret" {
		t.Errorf("BasicAuth = (%q, %q, %v), want (user, secret, true)", gotUser, gotPass, gotAuthOK)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
}

func TestDo_PostFormEncodes(t *testing.T) {
	var gotContentType, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, Username: "user", Password: "secret", HTTPClient: server.Client()}
	form := url.Values{"server_name": {"k3s lab"}, "type": {"hw"}}
	if _, err := c.do(context.Background(), http.MethodPost, "/reset/1", form); err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", gotContentType)
	}
	if gotBody != form.Encode() {
		t.Errorf("body = %q, want %q", gotBody, form.Encode())
	}
}

func TestDo_ErrorEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"status":404,"code":"SERVER_NOT_FOUND","message":"Server not found"}}`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	_, err := c.do(context.Background(), http.MethodGet, "/server/1", nil)
	if !IsNotFound(err) {
		t.Fatalf("err = %v, want a 404 APIError", err)
	}
	if !strings.Contains(err.Error(), "SERVER_NOT_FOUND") {
		t.Errorf("err = %q, want code SERVER_NOT_FOUND", err.Error())
	}
}

func TestDo_NonJSONErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`<html>502 Bad Gateway</html>`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	_, err := c.do(context.Background(), http.MethodGet, "/server", nil)
	if err == nil {
		t.Fatal("expected error for 502")
	}
	if !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "Bad Gateway") {
		t.Errorf("err = %q, want raw body fallback", err.Error())
	}
}

func TestDo_ConnectionError(t *testing.T) {
	c := &Client{Endpoint: "http://127.0.0.1:1", HTTPClient: &http.Client{}}
	_, err := c.do(context.Background(), http.MethodGet, "/server", nil)
	if err == nil || !strings.Contains(err.Error(), "executing request") {
		t.Fatalf("err = %v, want executing request error", err)
	}
}

func TestGetServer_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server/321" {
			t.Errorf("path = %q, want /server/321", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"server":{"server_number":321,"server_ip":"123.123.123.123","server_ipv6_net":"2a01:4f8:111:4221::","server_name":"server1","product":"DS 3000","dc":"NBG1-DC1","traffic":"5 TB","status":"ready","cancelled":false,"paid_until":"2010-09-02","ip":["123.123.123.123"],"subnet":[{"ip":"2a01:4f8:111:4221::","mask":"64"}]}}`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	s, err := c.GetServer(context.Background(), 321)
	if err != nil {
		t.Fatalf("GetServer returned error: %v", err)
	}
	if s.ServerNumber != 321 {
		t.Errorf("ServerNumber = %d, want 321", s.ServerNumber)
	}
	if s.ServerIP != "123.123.123.123" {
		t.Errorf("ServerIP = %q, want 123.123.123.123", s.ServerIP)
	}
	if s.Product != "DS 3000" || s.DC != "NBG1-DC1" || s.Status != "ready" {
		t.Errorf("unexpected facts: %+v", s)
	}
	if len(s.IPs) != 1 || s.IPs[0] != "123.123.123.123" {
		t.Errorf("IPs = %v, want [123.123.123.123]", s.IPs)
	}
	if len(s.Subnets) != 1 || s.Subnets[0].Mask != "64" {
		t.Errorf("Subnets = %v, want one /64", s.Subnets)
	}
}

func TestGetServer_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"status":404,"code":"SERVER_NOT_FOUND","message":"Server not found"}}`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	_, err := c.GetServer(context.Background(), 999)
	if !IsNotFound(err) {
		t.Fatalf("err = %v, want 404 APIError", err)
	}
}

func TestGetServer_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	_, err := c.GetServer(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "decoding server response") {
		t.Fatalf("err = %v, want decode error", err)
	}
}

func TestListServers_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"server":{"server_number":321,"server_ip":"1.1.1.1"}},{"server":{"server_number":322,"server_ip":"2.2.2.2"}}]`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	servers, err := c.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers returned error: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("len(servers) = %d, want 2", len(servers))
	}
	if servers[0].ServerNumber != 321 || servers[1].ServerNumber != 322 {
		t.Errorf("server numbers = %d,%d want 321,322", servers[0].ServerNumber, servers[1].ServerNumber)
	}
}

func TestListServers_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	_, err := c.ListServers(context.Background())
	if err == nil || !strings.Contains(err.Error(), "decoding server list response") {
		t.Fatalf("err = %v, want decode error", err)
	}
}

func TestServerByIP(t *testing.T) {
	body := `[{"server":{"server_number":321,"server_ip":"1.1.1.1","ip":["1.1.1.1","10.0.0.5"]}},{"server":{"server_number":322,"server_ip":"2.2.2.2","ip":["2.2.2.2"]}}]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}

	byPrimary, err := c.ServerByIP(context.Background(), "2.2.2.2")
	if err != nil {
		t.Fatalf("ServerByIP(primary) error: %v", err)
	}
	if byPrimary.ServerNumber != 322 {
		t.Errorf("ServerNumber = %d, want 322", byPrimary.ServerNumber)
	}

	bySecondary, err := c.ServerByIP(context.Background(), "10.0.0.5")
	if err != nil {
		t.Fatalf("ServerByIP(secondary) error: %v", err)
	}
	if bySecondary.ServerNumber != 321 {
		t.Errorf("ServerNumber = %d, want 321", bySecondary.ServerNumber)
	}

	_, err = c.ServerByIP(context.Background(), "9.9.9.9")
	if !IsNotFound(err) {
		t.Fatalf("err = %v, want 404 APIError for unknown IP", err)
	}
}

func TestCreateKey(t *testing.T) {
	var gotMethod, gotPath, gotName, gotData string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = r.ParseForm()
		gotName = r.PostForm.Get("name")
		gotData = r.PostForm.Get("data")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"key":{"name":"laptop","fingerprint":"aa:bb:cc","type":"ED25519","size":256,"data":"ssh-ed25519 AAAA laptop","created_at":"2026-06-26 10:00:00"}}`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	key, err := c.CreateKey(context.Background(), "laptop", "ssh-ed25519 AAAA laptop")
	if err != nil {
		t.Fatalf("CreateKey error: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/key" {
		t.Errorf("request = %s %s, want POST /key", gotMethod, gotPath)
	}
	if gotName != "laptop" || gotData != "ssh-ed25519 AAAA laptop" {
		t.Errorf("form = name=%q data=%q, want laptop / the key", gotName, gotData)
	}
	if key.Fingerprint != "aa:bb:cc" || key.Type != "ED25519" || key.Size != 256 {
		t.Errorf("key = %+v, want fingerprint aa:bb:cc type ED25519 size 256", key)
	}
}

func TestGetKey_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"status":404,"code":"NOT_FOUND","message":"key not found"}}`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	_, err := c.GetKey(context.Background(), "aa:bb:cc")
	if !IsNotFound(err) {
		t.Fatalf("err = %v, want 404 APIError", err)
	}
}

func TestUpdateKey(t *testing.T) {
	var gotPath, gotName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = r.ParseForm()
		gotName = r.PostForm.Get("name")
		_, _ = w.Write([]byte(`{"key":{"name":"workstation","fingerprint":"aa:bb:cc","type":"ED25519","size":256,"data":"ssh-ed25519 AAAA","created_at":"2026-06-26 10:00:00"}}`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	key, err := c.UpdateKey(context.Background(), "aa:bb:cc", "workstation")
	if err != nil {
		t.Fatalf("UpdateKey error: %v", err)
	}
	if gotPath != "/key/aa:bb:cc" {
		t.Errorf("path = %q, want /key/aa:bb:cc", gotPath)
	}
	if gotName != "workstation" {
		t.Errorf("name = %q, want workstation", gotName)
	}
	if key.Name != "workstation" {
		t.Errorf("key.Name = %q, want workstation", key.Name)
	}
}

func TestDeleteKey(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	if err := c.DeleteKey(context.Background(), "aa:bb:cc"); err != nil {
		t.Fatalf("DeleteKey error: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/key/aa:bb:cc" {
		t.Errorf("request = %s %s, want DELETE /key/aa:bb:cc", gotMethod, gotPath)
	}
}

func TestDeleteKey_PropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"status":500,"code":"KEY_DELETE_FAILED","message":"boom"}}`))
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL, HTTPClient: server.Client()}
	err := c.DeleteKey(context.Background(), "aa:bb:cc")
	if err == nil || !strings.Contains(err.Error(), "KEY_DELETE_FAILED") {
		t.Fatalf("err = %v, want KEY_DELETE_FAILED", err)
	}
}
