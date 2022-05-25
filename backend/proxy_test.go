package backend_test

import (
	"github.com/pkg/errors"
	"github.com/wandel/modprox/backend"
	"log"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestModuleProxy_GetList(t *testing.T) {
	b := backend.ModuleProxy{}

	if _, err := b.GetList("github.com/wandel/dne", ""); err != nil {
		if !errors.Is(err, backend.ErrNotFound) {
			t.Error("expected a ErrNotFound, got", err)
		}
	} else {
		t.Errorf("expected a ErrNotFound, got nil error")
	}

	if versions, err := b.GetList("github.com/wandel/modprox_test", ""); err != nil {
		t.Error("got unexpected error", err)
	} else {
		sort.Strings(versions)
		expected := []string{"v0.1.0", "v0.2.0", "v1.0.0", "v1.0.1"}
		if !reflect.DeepEqual(versions, expected) {
			t.Errorf("expected %s, got %s", expected, versions)
		}
	}

	if versions, err := b.GetList("github.com/wandel/modprox_test", "/v2"); err != nil {
		t.Error("got unexpected error", err)
	} else {
		sort.Strings(versions)
		expected := []string{"v2.0.0", "v2.1.0"}
		if !reflect.DeepEqual(versions, expected) {
			t.Errorf("expected %s, got %s", expected, versions)
		}
	}
}

func TestModuleProxy_GetLatest(t *testing.T) {
	b := backend.ModuleProxy{}

	if _, _, err := b.GetLatest("github.com/wandel/dne", ""); err != nil {
		if !errors.Is(err, backend.ErrNotFound) {
			t.Error("expected a ErrNotFound, got", err)
		}
	} else {
		t.Errorf("expected a ErrNotFound, got nil error")
	}

	if version, timestamp, err := b.GetLatest("github.com/wandel/modprox_test", ""); err != nil {
		t.Error("got unexpected error", err)
	} else {
		if version != "v1.0.1" {
			t.Errorf("expected version 'v1.0.1', got %s", version)
		}
		if timestamp.Format(time.RFC3339) != "2022-05-17T00:17:27Z" {
			t.Errorf("expected timestamp '2022-05-17T00:17:27Z', got %s", timestamp.Format(time.RFC3339))
		}
	}

	if version, timestamp, err := b.GetLatest("github.com/wandel/modprox_test", "/v2"); err != nil {
		t.Error("got unexpected error", err)
	} else {
		if version != "v2.1.0" {
			t.Errorf("expected version 'v2.1.0', got %s", version)
		}
		if timestamp.Format(time.RFC3339) != "2022-05-24T12:01:26Z" {
			t.Errorf("expected timestamp '2022-05-24T12:01:26Z', got %s", timestamp.Format(time.RFC3339))
		}
	}
}

func TestModuleProxy_GetModule(t *testing.T) {
	b := backend.ModuleProxy{}

	if _, err := b.GetModule("github.com/wandel/dne", "v1.0.0"); err != nil {
		if !errors.Is(err, backend.ErrNotFound) {
			t.Error("expected a ErrNotFound, got", err)
		}
	} else {
		t.Errorf("expected a ErrNotFound, got nil error")
	}

	if mod, err := b.GetModule("github.com/wandel/modprox_test", "v0.1.0"); err != nil {
		t.Error("got unexpected error", err)
	} else {
		expected := "module github.com/wandel/modprox_test"
		if mod != expected {
			t.Errorf("expected module result '%s', got '%s'", expected, mod)
		}
	}

	if mod, err := b.GetModule("github.com/wandel/modprox_test/v2", "v2.0.0"); err != nil {
		t.Error("got unexpected error", err)
	} else {
		expected := "module github.com/wandel/modprox_test/v2\n\ngo 1.18\n\nrequire github.com/pkg/errors v0.9.1 // indirect"
		if mod != expected {
			t.Errorf("expected version '%s', got '%s'", expected, mod)
		}
	}
}

func TestModuleProxy_GetInfo(t *testing.T) {
	b := backend.ModuleProxy{}

	if _, _, err := b.GetInfo("github.com/wandel/dne", "v1.0.0"); err != nil {
		if !errors.Is(err, backend.ErrNotFound) {
			t.Error("expected a ErrNotFound, got", err)
		}
	} else {
		t.Errorf("expected a ErrNotFound, got nil error")
	}

	if version, timestamp, err := b.GetInfo("github.com/wandel/modprox_test", "v0.1.0"); err != nil {
		t.Error("got unexpected error", err)
	} else {
		if version != "v0.1.0" {
			t.Errorf("expected version 'v0.1.0', got %s", version)
		}
		if timestamp.Format(time.RFC3339) != "2022-05-17T00:04:48Z" {
			t.Errorf("expected timestamp '2022-05-17T00:04:48Z', got %s", timestamp.Format(time.RFC3339))
		}
	}

	if version, timestamp, err := b.GetInfo("github.com/wandel/modprox_test/v2", "v2.0.0"); err != nil {
		t.Error("got unexpected error", err)
	} else {
		if version != "v2.0.0" {
			t.Errorf("expected version 'v2.0.0', got %s", version)
		}
		if timestamp.Format(time.RFC3339) != "2022-05-17T00:44:19Z" {
			t.Errorf("expected timestamp '2022-05-17T00:44:19Z', got %s", timestamp.Format(time.RFC3339))
		}
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

	if r, err := b.GetArchive("github.com/wandel/modprox_test/v2/", "v2.0.0"); err != nil {
		t.Error("got unexpected error", err)
	} else if resp, err := http.Get("https://proxy.golang.org/github.com/wandel/modprox_test/v2/@v/v2.0.0.zip"); err != nil {
		log.Fatalln("failed to download expected zip file")
	} else {
		CheckZips(resp.Body, r, t)
		resp.Body.Close()
	}
}
