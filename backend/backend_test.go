package backend_test

import (
	"archive/zip"
	"bytes"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"

	"golang.org/x/mod/module"

	"github.com/pkg/errors"
	"github.com/wandel/modprox/backend"
)

func NewZipReader(r io.Reader) (*zip.Reader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read content")
	}
	br := bytes.NewReader(data)
	return zip.NewReader(br, br.Size())
}

func CheckZips(r0 io.Reader, r1 io.Reader, t *testing.T) {
	zr0, err := NewZipReader(r0)
	if err != nil {
		t.Fatalf("failed to create zip reader for r0: %v", err)
	}

	zr1, err := NewZipReader(r1)
	if err != nil {
		t.Fatalf("failed to create zip reader for r1: %v", err)
	}

	for _, expected := range zr0.File {
		found := false
		for _, actual := range zr1.File {
			if expected.Name != actual.Name {
				continue
			}

			found = true
			if expected.CRC32 != actual.CRC32 {
				t.Errorf("expected file '%s' to have crc '%x', got '%x'", expected.Name, expected.CRC32, actual.CRC32)
			}

			if expected.UncompressedSize64 != actual.UncompressedSize64 {
				t.Errorf("expected file '%s' to have size '%d', but was '%d", expected.Name, expected.UncompressedSize64, actual.UncompressedSize64)
			}

			break
		}

		if !found {
			t.Errorf("missing file '%s'", expected.Name)
		}
	}

	for _, actual := range zr1.File {
		found := false
		for _, expected := range zr0.File {
			if actual.Name != expected.Name {
				continue
			}
			found = true
			// no need for size / crc32 checks here, they have already been done
			break
		}

		if !found {
			t.Errorf("extraneous file '%s'", actual.Name)
		}
	}
}

func CheckList(expected, actual backend.Backend, t *testing.T) {
	tests := []struct {
		path string
	}{
		{"github.com/wandel/dne"},
		{"github.com/wandel/modprox_test"},
		{"github.com/wandel/modprox_test/v2"},
		{"github.com/wandel/modprox_test/v3"},
		{"github.com/wandel/modprox_test/subpackage"},
		{"github.com/wandel/modprox_test/subpackage/v2"},
		{"gopkg.in/cheggaaa/pb.v1"},
		//	 {"golang.org/x/sys"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			prefix, major, ok := module.SplitPathVersion(tt.path)
			if !ok {
				t.Fatalf("failed to split path")
			}

			tags0, err0 := expected.GetList(prefix, major)
			tags1, err1 := actual.GetList(prefix, major)

			if errors.Is(err0, backend.ErrNotFound) && errors.Is(err1, backend.ErrNotFound) {
			} else if err0 != err1 {
				t.Errorf("expected '%s', got '%s'", err0, err1)
			}

			sort.Strings(tags0)
			sort.Strings(tags1)
			if !reflect.DeepEqual(tags0, tags1) {
				t.Errorf("expected '%s', got '%s'", tags0, tags1)
			}
		})
	}
}

func CheckLatest(expected, actual backend.Backend, t *testing.T) {
	tests := []struct {
		path  string
		major string
	}{
		//{"github.com/wandel/dne", ""},
		//{"github.com/wandel/modprox_test", ""},
		//{"github.com/wandel/modprox_test", "/v2"},
		//{"github.com/wandel/modprox_test/subpackage", ""},
		{"gopkg.in/cheggaaa/pb", ".v1"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			v0, ts0, err0 := expected.GetLatest(tt.path, tt.major)
			v1, ts1, err1 := actual.GetLatest(tt.path, tt.major)

			if errors.Is(err0, backend.ErrNotFound) && errors.Is(err1, backend.ErrNotFound) {
			} else if err0 != err1 {
				t.Errorf("expected '%s', got '%s'", err0, err1)
			}

			if v0 != v1 {
				t.Errorf("expected version '%s', got '%s'", v0, v1)

			}

			if ts0 != ts1 {
				t.Errorf("expected timestamp '%s', got '%s'", ts0, ts1)

			}
		})
	}
}

func CheckModule(expected, actual backend.Backend, t *testing.T) {
	tests := []struct {
		path    string
		version string
	}{
		//{"github.com/wandel/dne", "v1.0.0"},
		//{"github.com/wandel/modprox_test", "v0.2.0"},
		//{"github.com/wandel/modprox_test", "v1.0.0"},
		//{"github.com/wandel/modprox_test/v2", "v2.0.0"},
		//{"github.com/wandel/modprox_test/subpackage", "v0.1.0"},
		//{"github.com/wandel/modprox_test/subpackage", "v1.0.0"},
		//{"github.com/wandel/modprox_test/subpackage/v2", "v2.0.0"},
		//{"golang.org/x/sys", "v0.0.0-20211216021012-1d35b9e2eb4e"},
		//{"github.com/googleapis/gax-go/v2", "v2.1.1"},
		//{"go.opentelemetry.io/proto/otlp", "v0.7.0"},
		//{"gopkg.in/cheggaaa/pb.v1", "v1.0.27"},
		//{"github.com/Microsoft/go-winio", "v0.4.16"},
		//{"github.com/Azure/azure-sdk-for-go", "v64.0.0+incompatible"},
		//{"github.com/hashicorp/packer-plugin-virtualbox", "v1.0.4"},
		{"github.com/scaleway/packer-plugin-scaleway", "v1.0.4"},
	}
	for _, tt := range tests {
		t.Run(tt.path+"@"+tt.version, func(t *testing.T) {
			m0, err0 := expected.GetModule(tt.path, tt.version)
			m1, err1 := actual.GetModule(tt.path, tt.version)

			if errors.Is(err0, backend.ErrNotFound) && errors.Is(err1, backend.ErrNotFound) {
			} else if err0 != err1 {
				t.Errorf("expected '%s', got '%s'", err0, err1)
			}

			if strings.TrimSpace(m0) != strings.TrimSpace(m1) {
				t.Errorf("expected module '%s', got '%s'", m0, m1)
			}
		})
	}
}

func CheckInfo(expected, actual backend.Backend, t *testing.T) {
	tests := []struct {
		path    string
		version string
	}{
		{"github.com/wandel/dne", "v1.0.0"},
		{"github.com/wandel/modprox_test", "v1.0.1"},
		{"github.com/wandel/modprox_test/v2", "v2.0.0"},
		{"github.com/wandel/modprox_test/subpackage", "v1.0.0"},
		{"golang.org/x/sys", "v0.0.0-20211216021012-1d35b9e2eb4e"},
		{"github.com/googleapis/gax-go/v2", "v2.1.1"},
		{"github.com/scaleway/packer-plugin-scaleway", "v1.0.4"},
		{"go.opentelemetry.io/proto/otlp", "v0.7.0"},
		{"gopkg.in/cheggaaa/pb.v1", "v1.0.27"},
	}
	for _, tt := range tests {
		t.Run(tt.path+"@"+tt.version, func(t *testing.T) {
			v0, ts0, err0 := expected.GetInfo(tt.path, tt.version)
			v1, ts1, err1 := actual.GetInfo(tt.path, tt.version)

			if errors.Is(err0, backend.ErrNotFound) && errors.Is(err1, backend.ErrNotFound) {
			} else if err0 != err1 {
				t.Errorf("expected '%s', got '%s'", err0, err1)
			}

			if v0 != v1 {
				t.Errorf("expected version '%s', got '%s'", v0, v1)

			}

			if ts0 != ts1 {
				t.Errorf("expected timestamp '%s', got '%s'", ts0, ts1)
			}
		})
	}
}

func CheckArchive(expected, actual backend.Backend, t *testing.T) {
	tests := []struct {
		path    string
		version string
	}{
		{"github.com/wandel/dne", "v1.0.0"},
		{"github.com/wandel/modprox_test", "v1.0.1"},
		{"github.com/wandel/modprox_test/v2", "v2.0.0"},
		{"github.com/wandel/modprox_test/subpackage", "v1.0.0"},
		{"github.com/antihax/optional", "v1.0.0"},
		{"golang.org/x/sys", "v0.0.0-20211216021012-1d35b9e2eb4e"},
		{"github.com/googleapis/gax-go/v2", "v2.1.1"},
		{"github.com/scaleway/packer-plugin-scaleway", "v1.0.4"},
		{"go.opentelemetry.io/proto/otlp", "v0.7.0"},
		{"gopkg.in/cheggaaa/pb.v1", "v1.0.27"},
	}
	for _, tt := range tests {
		t.Run(tt.path+"@"+tt.version, func(t *testing.T) {
			r0, err0 := expected.GetArchive(tt.path, tt.version)
			r1, err1 := actual.GetArchive(tt.path, tt.version)

			if errors.Is(err0, backend.ErrNotFound) && errors.Is(err1, backend.ErrNotFound) {
			} else if err0 != err1 {
				t.Errorf("expected '%s', got '%s'", err0, err1)
			} else if r0 != nil && r1 != nil {
				CheckZips(r0, r1, t)
			}
		})
	}
}
