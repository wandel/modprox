package backend

import (
	"errors"
	"golang.org/x/exp/maps"
	"io"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
)

type Backend interface {
	GetList(path, major string) ([]string, error)
	GetLatest(path, major string) (string, time.Time, error)
	GetModule(path, version string) (string, error)
	GetInfo(path, version string) (string, time.Time, error)
	GetArchive(path, version string) (io.Reader, error)
}

type MultiBackend struct {
	backends []Backend
}

func NewMultiBackend(bs ...Backend) Backend {
	return &MultiBackend{
		backends: bs,
	}
}

func (mb MultiBackend) GetList(path, major string) ([]string, error) {
	results := map[string]bool{}
	for _, backend := range mb.backends {
		versions, err := backend.GetList(path, major)
		if err != nil {
			continue
		}

		for _, version := range versions {
			results[version] = true
		}
	}

	if len(results) == 0 {
		return nil, ErrNotFound
	}

	return maps.Keys(results), nil
}

func (mb MultiBackend) GetLatest(path, major string) (string, time.Time, error) {
	// git ls-remote -q origin
	version := ""
	timestamp := time.Unix(0, 0)
	for _, backend := range mb.backends {
		v, ts, err := backend.GetLatest(path, major)
		if err != nil {
			continue
		}

		if ts.After(timestamp) {
			version = v
			timestamp = ts
		}
	}

	if version == "" {
		return "", time.Unix(0, 0), ErrNotFound
	}

	return version, timestamp, nil
}

func (mb MultiBackend) GetModule(path, version string) (string, error) {
	for _, backend := range mb.backends {
		module, err := backend.GetModule(path, version)
		if err == nil {
			return module, nil
		}
	}
	return "", ErrNotFound
}

func (mb MultiBackend) GetInfo(path, version string) (string, time.Time, error) {
	for _, backend := range mb.backends {
		v, ts, err := backend.GetInfo(path, version)
		if err == nil {
			return v, ts, nil
		}
	}
	return "", time.Unix(0, 0), ErrNotFound
}

func (mb MultiBackend) GetArchive(path, version string) (io.Reader, error) {
	for _, backend := range mb.backends {
		r, err := backend.GetArchive(path, version)
		if err == nil {
			return r, nil
		}
	}
	return nil, ErrNotFound
}
