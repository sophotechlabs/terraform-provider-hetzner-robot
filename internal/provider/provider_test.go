package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestNew_ReturnsFactory(t *testing.T) {
	factory := New("1.0.0")
	if factory == nil {
		t.Fatal("New should not return nil")
	}

	p := factory()
	hp, ok := p.(*HrobotProvider)
	if !ok {
		t.Fatal("factory should return *HrobotProvider")
	}
	if hp.version != "1.0.0" {
		t.Errorf("version = %q, want %q", hp.version, "1.0.0")
	}
}

func TestHrobotProvider_Metadata(t *testing.T) {
	p := &HrobotProvider{version: "1.0.0"}
	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)

	if resp.TypeName != "hrobot" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "hrobot")
	}
	if resp.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", resp.Version, "1.0.0")
	}
}

func TestHrobotProvider_Schema_Attributes(t *testing.T) {
	p := &HrobotProvider{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	expected := []string{"username", "password", "endpoint"}
	for _, name := range expected {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("schema missing attribute %q", name)
		}
	}
	if len(resp.Schema.Attributes) != len(expected) {
		t.Errorf("schema has %d attributes, want %d", len(resp.Schema.Attributes), len(expected))
	}
}

func TestHrobotProvider_Schema_AllOptional(t *testing.T) {
	p := &HrobotProvider{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	for name, attr := range resp.Schema.Attributes {
		if !attr.IsOptional() {
			t.Errorf("attribute %q should be optional (env var fallback / default)", name)
		}
	}
}

func TestHrobotProvider_Schema_PasswordSensitive(t *testing.T) {
	p := &HrobotProvider{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	if !resp.Schema.Attributes["password"].IsSensitive() {
		t.Error("password attribute should be sensitive")
	}
	if resp.Schema.Attributes["username"].IsSensitive() {
		t.Error("username attribute should not be sensitive")
	}
}

func TestHrobotProvider_Resources(t *testing.T) {
	p := &HrobotProvider{}
	rs := p.Resources(context.Background())
	if len(rs) != 1 {
		t.Fatalf("len(Resources) = %d, want 1", len(rs))
	}
	if _, ok := rs[0]().(*sshKeyResource); !ok {
		t.Error("first resource should be *sshKeyResource")
	}
}

func TestHrobotProvider_DataSources(t *testing.T) {
	p := &HrobotProvider{version: "test"}
	ds := p.DataSources(context.Background())
	if len(ds) != 1 {
		t.Fatalf("len(DataSources) = %d, want 1", len(ds))
	}
	if _, ok := ds[0]().(*serverDataSource); !ok {
		t.Error("first data source should be *serverDataSource")
	}
}
