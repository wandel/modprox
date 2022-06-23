package backend_test

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"reflect"
	"sort"
	"strings"
	"testing"

	"golang.org/x/mod/module"

	"github.com/pkg/errors"
	"github.com/wandel/modprox/backend"
)

type testModule struct {
	path    string
	version string
}

func getTests() []testModule {
	return []testModule{
		//{"github.com/wandel/dne", "v1.0.0"},
		//{"github.com/wandel/modprox_test", "v1.0.1"},
		//{"github.com/wandel/modprox_test/v2", "v2.0.0"},
		//{"github.com/wandel/modprox_test/subpackage", "v1.0.0"},
		//{"golang.org/x/sys", "v0.0.0-20220622161953-175b2fd9d664"},
		//{"github.com/scaleway/packer-plugin-virtualbox", "v1.0.4"},
		//{"go.opentelemetry.io/proto/otlp", "v0.7.0"},
		//{"gopkg.in/yaml.v3", "v3.0.1"},
		//{"github.com/Azure/go-autorest", "v14.2.0+incompatible"},

		{"gopkg.in/inf.v0", "v0.9.1"},
		{"gopkg.in/ini.v1", "v1.66.2"},
		{"gopkg.in/check.v1", "v1.0.0-20190902080502-41f04d3bba15"},
		{"gopkg.in/check.v1", "v1.0.0-20180628173108-788fd7840127"},
		{"gopkg.in/cheggaaa/pb.v1", "v1.0.27"},
		{"gopkg.in/check.v1", "v0.0.0-20161208181325-20d25e280405"},
		{"gopkg.in/cheggaaa/pb.v1", "v1.0.25"},
		{"gopkg.in/check.v1", "v1.0.0-20200227125254-8fa46927fb4f"},
		{"gopkg.in/yaml.v2", "v2.0.0-20170812160011-eb3733d160e7"},
		{"gopkg.in/tomb.v1", "v1.0.0-20141024135613-dd632973f1e7"},
		{"gopkg.in/alecthomas/kingpin.v2", "v2.2.6"},
		{"gopkg.in/fsnotify.v1", "v1.4.7"},
	}
}

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
	for _, tt := range getTests() {
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

			if len(tags0) != 0 || len(tags1) != 0 {
				log.Println("[LIST]", tags0, len(tags0), tags1, len(tags1))
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
	for _, tt := range getTests() {
		t.Run(tt.path, func(t *testing.T) {
			prefix, major, ok := module.SplitPathVersion(tt.path)
			if !ok {
				t.Fatal("failed to split path", module.CheckPath(tt.path))
			}

			v0, ts0, err0 := expected.GetLatest(prefix, major)
			v1, ts1, err1 := actual.GetLatest(prefix, major)

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
	for _, tt := range getTests() {
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
	for _, tt := range getTests() {
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
	for _, tt := range getTests() {
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
