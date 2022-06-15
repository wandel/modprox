package utils_test

import (
	"github.com/wandel/modprox/utils"
	"testing"
)

func TestMapPath(t *testing.T) {
	tests := []struct {
		input string
		path  string
		major string
	}{
		{"gopkg.in/go-yaml/yaml.v0", "github.com/go-yaml/yaml", ""},
		{"gopkg.in/go-yaml/yaml.v1", "github.com/go-yaml/yaml", ""},
		{"gopkg.in/go-yaml/yaml.v2", "github.com/go-yaml/yaml", "v2"},
		{"gopkg.in/yaml.v1", "github.com/go-yaml/yaml", ""},
		{"gopkg.in/yaml.v2", "github.com/go-yaml/yaml", "v2"},
		{"golang.org/x/tools", "github.com/golang/tools", ""},
		{"golang.org/x/crypto", "github.com/golang/crypto", ""},
		{"google.golang.org/grpc", "github.com/grpc/grpc-go", ""},
		{"google.golang.org/protobuf", "github.com/protocolbuffers/protobuf-go", ""},
		{"google.golang.org/api", "github.com/googleapis/google-api-go-client", ""},
		{"google.golang.org/genproto", "github.com/googleapis/go-genproto", ""},
		{"go.opentelemetry.io/proto/otlp", "github.com/open-telemetry/opentelemetry-proto-go/otlp", ""},
		{"google.golang.org/appengine", "github.com/golang/appengine", ""},
		{"honnef.co/go/tools", "github.com/dominikh/go-tools", ""},
		{"honnef.co/go/tools/v2", "github.com/dominikh/go-tools", "v2"},
		{"cloud.google.com/go/vision/v2", "github.com/googleapis/google-cloud-go/vision", "v2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			path, major, err := utils.MapPath(tt.input)
			if err != nil {
				t.Fatal(err.Error())
			}
			if tt.path != path {
				t.Errorf("expected path %s, got %s", tt.path, path)
			}
			if tt.major != major {
				t.Errorf("expected major %s, got %s", tt.major, major)

			}
		})
	}
}
