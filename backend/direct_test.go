package backend_test

import (
	"github.com/wandel/modprox/backend"
	"testing"
)

var actual *backend.Direct

func init() {
	actual = &backend.Direct{}
	actual.Load("c:\\temp\\cache")
}

func TestDirect_GetList(t *testing.T) {
	expected := &backend.ModuleProxy{}
	CheckList(expected, actual, t)
}

func TestDirect_GetLatest(t *testing.T) {
	expected := &backend.ModuleProxy{}

	CheckLatest(expected, actual, t)
}

func TestDirect_GetModule(t *testing.T) {
	expected := &backend.ModuleProxy{}

	CheckModule(expected, actual, t)
}

func TestDirect_GetInfo(t *testing.T) {
	expected := &backend.ModuleProxy{}

	CheckInfo(expected, actual, t)
}

func TestDirect_GetArchive(t *testing.T) {
	expected := &backend.ModuleProxy{}

	CheckArchive(expected, actual, t)
}

func TestDirect_RepoCache(t *testing.T) {
	direct := &backend.Direct{}
	if _, err := direct.GetModule("github.com/wandel/modprox_test", "v1.0.0"); err != nil {
		t.Errorf("v1 should have worked: %v\n", err)
	}

	if _, err := direct.GetModule("github.com/wandel/modprox_test/v2", "v2.0.1"); err != nil {
		t.Errorf("v2 should have worked: %v\n", err)
	}

	if _, err := direct.GetModule("github.com/wandel/modprox_test/subpackage", "v1.0.0"); err != nil {
		t.Errorf("subpackage should have worked: %v\n", err)
	}

	t.Fail()
}
