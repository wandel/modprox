package backend_test

import (
	"testing"

	"github.com/wandel/modprox/backend"
)

func TestGitlab_GetList(t *testing.T) {
	expected := backend.ModuleProxy{}
	actual, err := backend.NewGitLab("", "mirror8")
	if err != nil {
		t.Fatal("failed to create gitlab backend", err)
	}

	CheckList(expected, actual, t)
}

func TestGitlab_GetLatest(t *testing.T) {
	expected := backend.ModuleProxy{}
	actual, err := backend.NewGitLab("", "mirror8")
	if err != nil {
		t.Fatal("failed to create gitlab backend", err)
	}

	CheckLatest(expected, actual, t)
}

func TestGitlab_GetModule(t *testing.T) {
	expected := backend.ModuleProxy{}
	actual, err := backend.NewGitLab("", "mirror8")
	if err != nil {
		t.Fatal("failed to create gitlab backend", err)
	}

	CheckModule(expected, actual, t)
}

func TestGitlab_GetInfo(t *testing.T) {
	expected := backend.ModuleProxy{}
	actual, err := backend.NewGitLab("", "mirror8")
	if err != nil {
		t.Fatal("failed to create gitlab backend", err)
	}

	CheckInfo(expected, actual, t)
}

func TestGitlab_GetArchive(t *testing.T) {
	expected := backend.ModuleProxy{}
	actual, err := backend.NewGitLab("", "mirror8")
	if err != nil {
		t.Fatal("failed to create gitlab backend", err)
	}

	CheckArchive(expected, actual, t)
}
