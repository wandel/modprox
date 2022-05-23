package backend

import (
	"archive/zip"
	"bytes"
	"fmt"
	"golang.org/x/exp/maps"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/mod/module"
)

type GitLab struct {
	host  string
	token string
	group string
}

func NewGitLab(token string) Backend {
	return &GitLab{
		token: token,
	}
}

func (b GitLab) GetList(path, major string) ([]string, error) {
	client, err := gitlab.NewClient(b.token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to gitlab")
	}

	// sync repository
	// TODO trigger and wait for a sync

	// fetch a list of tags
	tags, resp, err := client.Tags.ListTags("mirror8/"+path, &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 1000, // TODO should do this properly in the future
		},
	}, nil)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, errors.Wrap(err, "failed to get a list of tags")
	}

	// filter tag based on version
	tmp := map[string]bool{}
	for _, tag := range tags {
		log.Println(tag.Name)
		// go helpfully provides a simple function to validate a path and version (tag) combination.
		if err := module.CheckPathMajor(tag.Name, major); err == nil {
			tmp[module.CanonicalVersion(tag.Name)] = true
		}
	}

	results := maps.Keys(tmp)
	sort.Strings(results)
	return results, nil
}

func (b GitLab) GetLatest(path, major string) (string, time.Time, error) {
	client, err := gitlab.NewClient(b.token)
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to connect to gitlab")
	}

	// sync repository
	// TODO trigger and wait for a sync

	// fetch a list of tags
	tags, resp, err := client.Tags.ListTags("mirror8/"+path, &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 1000, // TODO should do this properly in the future

		},
	}, nil)
	log.Println(resp.Request.URL, resp.StatusCode, resp.Status)

	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return "", time.Unix(0, 0), ErrNotFound
		}
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to get a list of tags")
	}

	// filter tag based on version
	var latest *gitlab.Tag
	for _, tag := range tags {
		// go helpfully provides a simple function to validate a path and version (tag) combination.
		if err := module.CheckPathMajor(tag.Name, major); err != nil {
			continue
		}

		if tag.Commit == nil || tag.Commit.CommittedDate == nil {
			continue
		}

		if latest == nil {
			latest = tag
			continue
		}

		if latest.Commit.CommittedDate.Before(*tag.Commit.CommittedDate) {
			latest = tag
		}
	}

	if latest == nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to find a valid version")
	}

	return latest.Name, latest.Commit.CommittedDate.UTC(), nil
}

func (b GitLab) GetModule(path, version string) (string, error) {
	name, major, ok := module.SplitPathVersion(path)
	if !ok {
		return "", errors.New("failed to split path/version")
	}

	if module.IsPseudoVersion(version) {
		if commit, err := module.PseudoVersionRev(version); err != nil {
			return "", errors.Wrap(err, "failed to extract revision from pseudo version")
		} else {
			version = commit
		}
	}

	client, err := gitlab.NewClient(b.token)
	if err != nil {
		return "", errors.Wrap(err, "failed to connect to gitlab")
	}

	content, resp, err := client.RepositoryFiles.GetRawFile("mirror8/"+name, "go.mod", &gitlab.GetRawFileOptions{
		Ref: gitlab.String(version),
	})
	if err != nil {
		if resp.StatusCode != http.StatusNotFound {
			// some other error occurred
			return "", errors.Wrap(err, "failed to fetch mod file")
		}

		if !strings.Contains(err.Error(), "404 File Not Found") {
			// gitlab will also 404 if the project does not exist
			return "", ErrNotFound
		}

		if major == "" {
			// generate a synthetic go.mod file if one does not exist (only appropriate for v0/v1)
			return fmt.Sprintf("module %s", path), nil
		}

		return "", ErrNotFound
	}

	return string(content), nil
}

func (b GitLab) GetInfo(path, version string) (string, time.Time, error) {
	name, _, ok := module.SplitPathVersion(path)
	if !ok {
		return "", time.Unix(0, 0), errors.New("failed to split path/version")
	}

	client, err := gitlab.NewClient(b.token)
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to connect to gitlab")
	}

	if tag, resp, err := client.Tags.GetTag("mirror8/"+name, version, nil); err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return "", time.Unix(0, 0), ErrNotFound
		}
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to get tag")
	} else {
		return tag.Name, tag.Commit.CommittedDate.UTC(), nil
	}

	return "", time.Unix(0, 0), ErrNotFound
}

func (b GitLab) GetArchive(path, version string) (io.Reader, error) {
	name, _, ok := module.SplitPathVersion(path)
	if !ok {
		return nil, errors.New("failed to split path/version")
	}

	if module.IsPseudoVersion(version) {
		if commit, err := module.PseudoVersionRev(version); err != nil {
			return nil, errors.Wrap(err, "Failed to parse pseudo version")
		} else {
			version = commit
		}
	}

	client, err := gitlab.NewClient(b.token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to gitlab")
	}

	var buffer bytes.Buffer
	if resp, err := client.Repositories.StreamArchive("mirror8/"+name, &buffer, &gitlab.ArchiveOptions{
		Format: gitlab.String("zip"),
		SHA:    gitlab.String(version),
	}, nil); err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, errors.Wrapf(err, "failed to get archive stream from gitlab: status=%d", resp.StatusCode)
	}

	reader := bytes.NewReader(buffer.Bytes())
	input, err := zip.NewReader(reader, reader.Size())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new zip reader")
	}

	if path, err = module.EscapePath(path); err != nil {
		return nil, errors.Wrap(err, "failed escape module path")
	}

	if version, err = module.EscapeVersion(version); err != nil {
		return nil, errors.Wrap(err, "failed to escape version")
	}

	base := path + "@" + version + "/"

	// we buffer the new zip file in memory so we can return useful http error codes/messages.
	// if we did zip.NewWriter(w) then as soon as the zip writer flushed data, the http server would
	// automatically send a http.StatusOK, preventing us from returning any context of the error.
	var buffer1 bytes.Buffer
	output := zip.NewWriter(&buffer1)
	var replace string
	for _, file := range input.File {
		if replace == "" {
			replace = file.Name
		}

		if strings.Contains(file.Name, "vendor") {
			if !strings.HasSuffix(file.Name, "modules.txt") {
				// we leave out vendor files, except than modules.txt
				continue
			}
		}

		// need to replace the base folder name with the correct module path (ie "github.com/urfave/cli/v2@v2.6.0")
		tmp := strings.Replace(file.Name, replace, base, 1)
		if dst, err := output.Create(tmp); err != nil {
			return nil, errors.Wrap(err, "failed to create file in new zip file")
		} else if src, err := file.Open(); err != nil {
			return nil, errors.Wrap(err, "failed to open file")
		} else if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			return nil, errors.Wrap(err, "failed to copy file")
		} else {
			src.Close()
		}

	}
	output.Close()
	return &buffer1, nil
}
