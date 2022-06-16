package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/mod/module"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type ModuleProxy struct {
}

func (b ModuleProxy) GetList(path, major string) ([]string, error) {
	log.Println("list:", path, major)
	escaped, err := module.EscapePath(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to escape module path '%s'", path)
	}

	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/list", escaped+major)
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query module proxy")
	} else if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, ErrNotFound
	} else if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(err, "unexpected error from the module proxy")
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read body of response")
	}

	data = bytes.TrimSpace(data)
	versions := strings.Split(string(data), "\n")
	sort.Strings(versions)
	return versions, nil
}

func (b ModuleProxy) GetLatest(path, major string) (string, time.Time, error) {
	log.Println("latest:", path, major)
	escaped, err := module.EscapePath(path)
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrapf(err, "failed to escape module path '%s'", path)
	}
	url := fmt.Sprintf("https://proxy.golang.org/%s/@latest", escaped+major)
	resp, err := http.Get(url)
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to query module proxy")
	} else if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return "", time.Unix(0, 0), ErrNotFound
	} else if resp.StatusCode != http.StatusOK {
		return "", time.Unix(0, 0), errors.Wrap(err, "unexpected error from the module proxy")
	}
	defer resp.Body.Close()

	var data struct {
		Version string
		Time    time.Time
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to json decode result")
	}

	return data.Version, data.Time, nil
}

func (b ModuleProxy) GetModule(path, version string) (string, error) {
	log.Println("module:", path, version)
	escaped, err := module.EscapePath(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to escape module path '%s'", path)
	}
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.mod", escaped, version)
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "failed to query module proxy")
	} else if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return "", ErrNotFound
	} else if resp.StatusCode != http.StatusOK {
		return "", errors.Wrap(err, "unexpected error from the module proxy")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read body of response")
	}

	data = bytes.TrimSpace(data)
	return string(data), nil
}

func (b ModuleProxy) GetInfo(path, version string) (string, time.Time, error) {
	log.Println("info:", path, version)
	escaped, err := module.EscapePath(path)
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrapf(err, "failed to escape module path '%s'", path)
	}
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.info", escaped, version)
	resp, err := http.Get(url)
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to query module proxy")
	} else if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return "", time.Unix(0, 0), ErrNotFound
	} else if resp.StatusCode != http.StatusOK {
		return "", time.Unix(0, 0), errors.Wrap(err, "unexpected error from the module proxy")
	}
	defer resp.Body.Close()

	var data struct {
		Version string
		Time    time.Time
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to json decode result")
	}

	return data.Version, data.Time, nil
}

func (b ModuleProxy) GetArchive(path, version string) (io.Reader, error) {
	log.Println("archive:", path, version)
	escaped, err := module.EscapePath(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to escape module path '%s'", path)
	}

	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.zip", escaped, version)
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query module proxy")
	} else if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, ErrNotFound
	} else if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(err, "unexpected error from the module proxy")
	}
	defer resp.Body.Close()

	var archive bytes.Buffer
	if _, err := io.Copy(&archive, resp.Body); err != nil {
		return nil, errors.Wrap(err, "failed to buffer archive in memory")
	}

	return &archive, nil
}
