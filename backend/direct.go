package backend

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Direct struct {
	Environ []string
}

type moduleInfo struct {
	Version  string   // module version
	Versions []string // available module versions (with -versions)
	Time     string   // time version was created
	Dir      string   // directory holding files for this module, if any
	Zip      string   // absolute path to cached .zip file
	GoMod    string   // path to go.mod file for this module, if any
	Error    string   // error loading module
}

func (d *Direct) execute(name string, args ...string) (moduleInfo, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	for _, value := range d.Environ {
		cmd.Env = append(cmd.Env, value)
	}

	stdout := bytes.Buffer{}
	cmd.Stdout = &stdout
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr

	cmd.Run()
	//log.Println("shell:", cmd.Args)
	//log.Println("stdout:", stdout.String())
	//log.Println("stderr:", stderr.String())

	var info moduleInfo
	if stdout.Len() == 0 {
		info.Error = stderr.String()
	} else if err := json.NewDecoder(&stdout).Decode(&info); err != nil {
		return info, errors.Wrap(err, "failed to decode stdout")
	}

	return info, nil
}

func (d *Direct) GetList(path, major string) ([]string, error) {
	info, err := d.execute("go", "list", "-retracted", "-x", "-m", "-versions", "-json", path+major)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read output")
	}

	if strings.Contains(info.Error, "Repository not found") {
		return nil, ErrNotFound
	} else if len(info.Versions) == 0 && info.Version == "" {
		return nil, ErrNotFound
	}

	return info.Versions, nil
}

func (d *Direct) GetLatest(path, major string) (string, time.Time, error) {
	return d.GetInfo(path+major, "latest")
}

func (d *Direct) GetInfo(path, version string) (string, time.Time, error) {
	info, err := d.execute("go", "list", "-retracted", "-x", "-m", "-json", path+"@"+version)
	if err != nil {
		log.Println("error:", err)
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to run go list")
	}

	if strings.Contains(info.Error, "Repository not found") {
		log.Println("error:", info)
		return "", time.Unix(0, 0), ErrNotFound
	} else if strings.Contains(info.Error, "unknown revision") {
		log.Println("error:", info)
		return "", time.Unix(0, 0), ErrOutOfDate
	}

	ts, err := time.Parse(time.RFC3339, info.Time)
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to parse the timestamp")
	}

	return info.Version, ts, nil
}

func (d *Direct) GetModule(path, version string) (string, error) {
	info, err := d.execute("go", "list", "-retracted", "-x", "-m", "-json", path+"@"+version)
	if err != nil {
		return "", errors.Wrap(err, "failed to run go list")
	}

	if strings.Contains(info.Error, "Repository not found") {
		return "", ErrNotFound
	} else if strings.Contains(info.Error, "unknown revision") {
		return "", ErrOutOfDate
	}

	f, err := os.Open(info.GoMod)
	if err != nil {
		return "", errors.Wrapf(err, "failed to open '%s'\n", info.GoMod)
	}
	defer f.Close()

	mod, err := io.ReadAll(f)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read '%s'", info.GoMod)
	}

	return string(mod), nil
}

func (d *Direct) GetArchive(path, version string) (io.Reader, error) {
	info, err := d.execute("go", "mod", "download", "-x", "-json", path+"@"+version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run go list")
	}

	if strings.Contains(info.Error, "Repository not found") {
		return nil, ErrNotFound
	} else if strings.Contains(info.Error, "unknown revision") {
		return nil, ErrOutOfDate
	}

	f, err := os.Open(info.Zip)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open '%s'\n", info.GoMod)
	}
	defer f.Close()

	zip, err := io.ReadAll(f)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read '%s'", info.GoMod)
	}

	return bytes.NewReader(zip), nil
}
