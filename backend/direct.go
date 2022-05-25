package backend

import (
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/pkg/errors"
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
