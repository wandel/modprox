package backend_test

import (
	"github.com/wandel/modprox/backend"
	"testing"
)

func TestGitlab_GetList(t *testing.T) {
	trusted := backend.ModuleProxy{}
	backend := backend.NewGitLab("")

	CheckList(trusted, backend, t)
}

func TestGitlab_GetLatest(t *testing.T) {
	trusted := backend.ModuleProxy{}
	backend := backend.NewGitLab("")

	CheckLatest(trusted, backend, t)
}

func TestGitlab_GetModule(t *testing.T) {
	trusted := backend.ModuleProxy{}
	backend := backend.NewGitLab("")

	CheckModule(trusted, backend, t)
}

func TestGitlab_GetInfo(t *testing.T) {
	trusted := backend.ModuleProxy{}
	backend := backend.NewGitLab("")

	CheckInfo(trusted, backend, t)
}

func TestGitlab_GetArchive(t *testing.T) {
	trusted := backend.ModuleProxy{}
	backend := backend.NewGitLab("")

	CheckArchive(trusted, backend, t)
}
