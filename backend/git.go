package backend

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"
	"golang.org/x/mod/module"
	"io"
	"time"
)

type Git struct {
	remote string
	auth   transport.AuthMethod
}

func NewGit(remote string, auth transport.AuthMethod) Backend {
	return Git{
		remote: remote,
		auth:   auth,
	}
}

func (b Git) GetList(path, major string) ([]string, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{b.remote},
	})

	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch tags")
	}

	var results []string
	for _, ref := range refs {
		name := ref.Name()
		if !name.IsTag() {
			continue
		}

		if err := module.CheckPathMajor(name.Short(), major); err != nil {
			continue
		}
		results = append(results, name.Short())
	}

	return nil, errors.New("Not Implemented Yet")
}

func (b Git) GetLatest(path, major string) (string, time.Time, error) {
	return "", time.Unix(0, 0), errors.New("Not Implemented Yet")
}

func (b Git) GetModule(path, version string) (string, error) {
	return "", errors.New("Not Implemented Yet")
}

func (b Git) GetInfo(path, version string) (string, time.Time, error) {
	return "", time.Unix(0, 0), errors.New("Not Implemented Yet")
}

func (b Git) GetArchive(path, version string) (io.Reader, error) {
	return nil, errors.New("Not Implemented Yet")
}
