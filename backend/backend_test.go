package backend_test

import (
	"archive/zip"
	"bytes"
	"github.com/pkg/errors"
	"github.com/wandel/modprox/backend"
	"io"
	"reflect"
	"testing"
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
}

func CheckList(expected, actual backend.Backend, t *testing.T) {
	tests := []struct {
		path    string
		version string
	}{
		{"github.com/wandel/dne", ""},
		{"github.com/wandel/modprox_test", ""},
		{"github.com/wandel/modprox_test", "/v2"},
	}
	for _, tt := range tests {
		t.Run(tt.path+tt.version, func(t *testing.T) {
			tags0, err0 := expected.GetList(tt.path, tt.version)
			tags1, err1 := actual.GetList(tt.path, tt.version)

			if errors.Is(err0, backend.ErrNotFound) && errors.Is(err1, backend.ErrNotFound) {
			} else if err0 != err1 {
				t.Errorf("expected '%s', got '%s'", err0, err1)
			}

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
		{"github.com/wandel/dne", ""},
		{"github.com/wandel/modprox_test", ""},
		{"github.com/wandel/modprox_test", "/v2"},
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
		{"github.com/wandel/dne", "v1.0.0"},
		{"github.com/wandel/modprox_test", "v0.2.0"},
		{"github.com/wandel/modprox_test", "v1.0.0"},
		{"github.com/wandel/modprox_test/v2", "v2.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.path+"@"+tt.version, func(t *testing.T) {
			m0, err0 := expected.GetModule(tt.path, tt.version)
			m1, err1 := actual.GetModule(tt.path, tt.version)

			if errors.Is(err0, backend.ErrNotFound) && errors.Is(err1, backend.ErrNotFound) {
			} else if err0 != err1 {
				t.Errorf("expected '%s', got '%s'", err0, err1)
			}

			if m0 != m1 {
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
	}
	for _, tt := range tests {
		t.Run(tt.path+"@"+tt.version, func(t *testing.T) {
			r0, err0 := expected.GetArchive(tt.path, tt.version)
			r1, err1 := actual.GetArchive(tt.path, tt.version)

			if errors.Is(err0, backend.ErrNotFound) && errors.Is(err1, backend.ErrNotFound) {
			} else if err0 != err1 {
				t.Errorf("expected '%s', got '%s'", err0, err1)
			} else if r0 != nil && r1 != nil {
				//Do I need to test this in reverse?
				CheckZips(r0, r1, t)
			}
		})
	}
}
