package backend_test

import (
	"log"
	"net/http"
	"reflect"
	"testing"
	"time"

	"golang.org/x/mod/module"

	"github.com/pkg/errors"
	"github.com/wandel/modprox/backend"
)

func TestModuleProxy_GetList(t *testing.T) {
	b := backend.ModuleProxy{}

	tests := []struct {
		path     string
		versions []string
	}{
		{"github.com/wandel/dne", nil},
		{"github.com/wandel/modprox_test", []string{"v0.1.0", "v0.2.0", "v1.0.0", "v1.0.1"}},
		{"github.com/wandel/modprox_test/v2", []string{"v2.0.0", "v2.1.0"}},
		{"github.com/wandel/modprox_test/v3", nil},
		{"github.com/wandel/modprox_test/subpackage", []string{"v0.1.0", "v1.0.0"}},
		{"github.com/wandel/modprox_test/subpackage/v2", nil},
		{"gopkg.in/cheggaaa/pb.v2", []string{"v2.0.0", "v2.0.1", "v2.0.2", "v2.0.3", "v2.0.4", "v2.0.5", "v2.0.6", "v2.0.7"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			prefix, major, ok := module.SplitPathVersion(tt.path)
			if !ok {
				t.Fatalf("failed to split path: %s", tt.path)
			}
			log.Println("[PROXY]", prefix, major)

			if versions, err := b.GetList(prefix, major); err != nil {
				if tt.versions != nil || !errors.Is(err, backend.ErrNotFound) {
					t.Fatal("expected a 404 not found error, got", err)
				}
			} else if !reflect.DeepEqual(tt.versions, versions) {
				t.Fatalf("expected '%s', got '%s'", tt.versions, versions)
			}
		})
	}
}

func TestModuleProxy_GetLatest(t *testing.T) {
	b := backend.ModuleProxy{}

	tests := []struct {
		path      string
		version   string
		timestamp string
	}{
		{"github.com/wandel/dne", "", ""},
		{"github.com/wandel/modprox_test", "v1.0.1", "2022-05-17T00:17:27Z"},
		{"github.com/wandel/modprox_test/v2", "v2.1.0", "2022-05-24T12:01:26Z"},
		{"github.com/wandel/modprox_test/v3", "", ""},
		{"github.com/wandel/modprox_test/subpackage", "v1.0.0", "2022-05-24T12:01:26Z"},
		{"github.com/wandel/modprox_test/subpackage/v2", "", ""},
		{"gopkg.in/cheggaaa/pb.v2", "v2.0.7", "2019-07-02T10:37:31Z"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			prefix, major, ok := module.SplitPathVersion(tt.path)
			if !ok {
				t.Fatalf("failed to split path: %s", tt.path)
			}

			version, timestamp, err := b.GetLatest(prefix, major)
			if tt.version == "" && tt.timestamp == "" {
				if !errors.Is(err, backend.ErrNotFound) {
					t.Fatal("expected a 404 not found error, got", err)
				}
			} else {
				if tt.version != version {
					t.Errorf("expected %s, got '%s'", tt.version, version)
				}

				if timestamp.Format(time.RFC3339) != tt.timestamp {
					t.Errorf("expected timestamp '%s', got '%s'", tt.timestamp, timestamp.Format(time.RFC3339))
				}
			}
		})
	}
}

func TestModuleProxy_GetModule(t *testing.T) {
	b := backend.ModuleProxy{}

	tests := []struct {
		path    string
		version string
		module  string
	}{
		{"github.com/wandel/dne", "v1.0.0", ""},
		{"github.com/wandel/modprox_test", "v0.1.0", "module github.com/wandel/modprox_test"},
		{"github.com/wandel/modprox_test", "v1.0.1", "module github.com/wandel/modprox_test\n\ngo 1.18"},
		{"github.com/wandel/modprox_test/v2", "v2.1.0", "module github.com/wandel/modprox_test/v2\n\ngo 1.18\n\nrequire github.com/pkg/errors v0.9.1 // indirect"},
		{"github.com/wandel/modprox_test/v3", "v3.0.0", ""},
		{"github.com/wandel/modprox_test/subpackage", "v0.1.0", ""},
		{"github.com/wandel/modprox_test/subpackage", "v1.0.0", "module github.com/wandel/modprox_test/subpackage\n\ngo 1.18"},
		{"github.com/wandel/modprox_test/subpackage/v2", "v2.0.0", ""},
		{"gopkg.in/cheggaaa/pb.v2", "v2.0.7", "module gopkg.in/cheggaaa/pb.v2"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			module, err := b.GetModule(tt.path, tt.version)
			if tt.module == "" {
				if !errors.Is(err, backend.ErrNotFound) {
					t.Fatal("expected a 404 not found error, got", err)
				}
			} else if tt.module != module {
				t.Errorf("expected module '%s', got '%s'", tt.module, module)
			}
		})
	}
}

func TestModuleProxy_GetInfo(t *testing.T) {
	b := backend.ModuleProxy{}

	tests := []struct {
		path      string
		version   string
		timestamp string
	}{
		{"github.com/wandel/dne", "v1.0.0", ""},
		{"github.com/wandel/modprox_test", "v1.0.1", "2022-05-17T00:17:27Z"},
		{"github.com/wandel/modprox_test/v2", "v2.1.0", "2022-05-24T12:01:26Z"},
		{"github.com/wandel/modprox_test/v3", "", ""},
		{"github.com/wandel/modprox_test/subpackage", "v0.1.0", ""},
		{"github.com/wandel/modprox_test/subpackage", "v1.0.0", "2022-05-24T12:01:26Z"},
		{"github.com/wandel/modprox_test/subpackage/v2", "", ""},
		{"gopkg.in/cheggaaa/pb.v2", "v2.0.7", "2019-07-02T10:37:31Z"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			version, timestamp, err := b.GetInfo(tt.path, tt.version)
			if tt.timestamp == "" {
				if !errors.Is(err, backend.ErrNotFound) {
					t.Fatal("expected a 404 not found error, got", err)
				}
			} else {
				if tt.version != version {
					t.Errorf("expected %s, got '%s'", tt.version, version)
				}

				if timestamp.Format(time.RFC3339) != tt.timestamp {
					t.Errorf("expected timestamp '%s', got '%s'", tt.timestamp, timestamp.Format(time.RFC3339))
				}
			}
		})
	}
}

func TestModuleProxy_GetArchive(t *testing.T) {
	b := backend.ModuleProxy{}

	if _, err := b.GetArchive("github.com/wandel/dne", "v1.0.0"); err != nil {
		if !errors.Is(err, backend.ErrNotFound) {
			t.Error("expected a ErrNotFound, got", err)
		}
	} else {
		t.Errorf("expected a ErrNotFound, got nil error")
	}

	if r, err := b.GetArchive("github.com/wandel/modprox_test", "v1.0.0"); err != nil {
		t.Error("got unexpected error", err)
	} else if resp, err := http.Get("https://proxy.golang.org/github.com/wandel/modprox_test/@v/v1.0.0.zip"); err != nil {
		log.Fatalln("failed to download expected zip file")
	} else {
		CheckZips(resp.Body, r, t)
		resp.Body.Close()
	}

	if r, err := b.GetArchive("github.com/wandel/modprox_test/v2", "v2.0.0"); err != nil {
		t.Error("got unexpected error", err)
	} else if resp, err := http.Get("https://proxy.golang.org/github.com/wandel/modprox_test/v2/@v/v2.0.0.zip"); err != nil {
		log.Fatalln("failed to download expected zip file")
	} else {
		CheckZips(resp.Body, r, t)
		resp.Body.Close()
	}
}
