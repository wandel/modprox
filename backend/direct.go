package backend

import (
	"github.com/pkg/errors"
	"io"
	"time"
)

type Download struct {
	Path     string
	Version  string
	Info     string
	GoMod    string
	Zip      string
	Dir      string
	Sum      string
	GoModSum string
}

type Direct struct{}

func (b Direct) GetList(path, major string) ([]string, error) {
	return nil, errors.New("Not Implemented Yet")
}

func (b Direct) GetLatest(path, major string) (string, time.Time, error) {
	return "", time.Unix(0, 0), errors.New("Not Implemented Yet")
}

func (b Direct) GetModule(path, version string) (string, error) {
	return "", errors.New("Not Implemented Yet")
}

func (b Direct) GetInfo(path, version string) (string, time.Time, error) {
	return "", time.Unix(0, 0), errors.New("Not Implemented Yet")
}

func (b Direct) GetArchive(path, version string) (io.Reader, error) {
	return nil, errors.New("Not Implemented Yet")
}
