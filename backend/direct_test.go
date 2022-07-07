package backend_test

import (
	"testing"

	"github.com/wandel/modprox/backend"
)

func init() {
}

func TestDirect_GetList(t *testing.T) {
	CheckList(&expected, &backend.Direct{}, t)
}

func TestDirect_GetLatest(t *testing.T) {
	CheckLatest(&expected, &backend.Direct{}, t)
}

func TestDirect_GetModule(t *testing.T) {
	CheckModule(&expected, &backend.Direct{}, t)
}

func TestDirect_GetInfo(t *testing.T) {
	CheckInfo(&expected, &backend.Direct{}, t)
}

func TestDirect_GetArchive(t *testing.T) {
	CheckArchive(&expected, &backend.Direct{}, t)
}
