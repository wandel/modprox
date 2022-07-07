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

type testModule struct {
	path    string
	version string
}

var expected backend.ModuleProxy

func getTests() []testModule {
	return []testModule{
		//// does not exist
		//{"github.com/wandel/dne", "v1.0.0"},
		//// no major version
		//{"github.com/wandel/modprox-test", "v1.0.1"},
		//// major version > 1
		//{"github.com/wandel/modprox-test/v2", "v2.0.0"},
		//// module in a subdirectory
		//{"github.com/wandel/modprox-test/subpackage", "v1.0.0"},
		//// module in a subdirectory does not exist
		//{"github.com/wandel/modprox-test/subpackage/v2", "v2.0.0"},
		//// gopkg.in support
		//{"gopkg.in/wandel/modprox-test.v0", "v0.2.0"},
		//// there is no mod file in v1.0.0, so this will fail
		////{"gopkg.in/wandel/modprox-test.v1", "v1.0.0"},
		//{"gopkg.in/wandel/modprox-test.v2", "v2.0.0"},
		//// no major tags
		//{"golang.org/x/sys", "v0.0.0-20220622161953-175b2fd9d664"},
		//// +incompatible version
		//{"github.com/Azure/go-autorest", "v14.2.0+incompatible"},
		//{"github.com/fsnotify/fsnotify", "v1.5.4"},
		//{"github.com/hashicorp/packer-plugin-vsphere", "v1.0.5"},
		//{"github.com/scaleway/packer-plugin-scaleway", "v1.0.4"},
		//{"cloud.google.com/go/storage", "v1.18.2"},
		//{"github.com/Azure/go-autorest/autorest", "v0.11.19"},
		//{"github.com/Azure/go-autorest/logger", "v0.2.1"},
		{"github.com/Azure/go-autorest/tracing", "v0.6.0"},
		//{"github.com/aliyun/alibaba-cloud-sdk-go", "v1.61.1028"},
		//{"github.com/hashicorp/consul/api", "v1.10.1"},
		//{"github.com/hashicorp/go-oracle-terraform", "v0.17.0"},
		//{"github.com/hashicorp/serf", "v0.9.5"},
		//{"github.com/joyent/triton-go", "v1.8.5"},
		//{"github.com/mailru/easyjson", "v0.7.6"},
		//{"github.com/ucloud/ucloud-sdk-go", "v0.20.2"},
		//{"go.mongodb.org/mongo-driver", "v1.5.1"},

		// this has beta versions that have no tag present in vcs... maybe deleted?
		//{"github.com/hashicorp/vault/api", "v1.1.1"},
		//{"github.com/ugorji/go/codec", "v1.2.6"},
		//{"github.com/kevinburke/ssh_config", "v0.0.0-20201106050909-4977a11b4351"},
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
				return
			} else if err0 != err1 {
				t.Errorf("expected '%s', got '%s'", err0, err1)
			}

			sort.Strings(tags0)
			sort.Strings(tags1)
			if !reflect.DeepEqual(tags0, tags1) {
				t.Errorf("expected '%+v', got '%+v'", tags0, tags1)
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

			if err0 != nil || err1 != nil {
				if !errors.Is(err0, backend.ErrNotFound) || !errors.Is(err1, backend.ErrNotFound) {
					t.Fatalf("expected '%s', got '%s'", err0, err1)
				}
			} else {
				if v0 != v1 {
					t.Errorf("expected version '%s', got '%s'", v0, v1)

				}

				if ts0 != ts1 {
					t.Errorf("expected timestamp '%s', got '%s'", ts0, ts1)
				}
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
