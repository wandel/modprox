package backend_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/wandel/modprox/backend"
)

var actual backend.Git

func init() {
	actual.CacheDir = "C:\\temp\\cache"
	actual.Load()
}

func TestGit_GetList(t *testing.T) {
	CheckList(&expected, &actual, t)
}

func TestGit_GetLatest(t *testing.T) {
	CheckLatest(&expected, &actual, t)
}

func TestGit_GetModule(t *testing.T) {
	CheckModule(&expected, &actual, t)
}

func TestGit_GetInfo(t *testing.T) {
	CheckInfo(&expected, &actual, t)
}

func TestGit_GetArchive(t *testing.T) {
	CheckArchive(&expected, &actual, t)
}

func TestGit_RepoCache(t *testing.T) {
	actual := &backend.Git{}
	if path, err := ioutil.TempDir("", "modprox-test-*"); err != nil {
		t.Fatal("failed to create temp cache directory", err)
	} else {
		defer os.RemoveAll(path)
		actual.CacheDir = path
	}

	if _, err := actual.GetModule("github.com/wandel/modprox-test", "v1.0.0"); err != nil {
		t.Errorf("v1 should have worked: %v\n", err)
	}

	if _, err := actual.GetModule("github.com/wandel/modprox-test/v2", "v2.0.0"); err != nil {
		t.Errorf("v2 should have worked: %v\n", err)
	}

	if _, err := actual.GetModule("github.com/wandel/modprox-test/subpackage", "v1.0.0"); err != nil {
		t.Errorf("subpackage should have worked: %v\n", err)
	}
}
