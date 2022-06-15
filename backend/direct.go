package backend

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/wandel/modprox/utils"
	"io"
	"log"
	path2 "path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	"golang.org/x/mod/module"
)

type Direct struct {
	cache sync.Map
	locks sync.Map
}

func splitSubModule(path, base string) string {
	return strings.TrimPrefix(strings.TrimPrefix(path, base), "/")
}

func ListRemote(path string) ([]*plumbing.Reference, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{
			fmt.Sprintf("https://%s", path),
		},
	})

	refs, err := remote.List(&git.ListOptions{})
	if errors.Is(err, transport.ErrRepositoryNotFound) || errors.Is(err, transport.ErrAuthenticationRequired) {
		log.Println("[MISSING]", path)
		return nil, ErrNotFound
	}
	return refs, err
}

func (b *Direct) getRepository(path string) (*git.Repository, string, error) {
	mapUrl := func(path string) string {
		if mapped, _, err := utils.MapPath(path); err != nil {
			return ""
		} else {
			return mapped
		}
	}

	for tmp := path; tmp != "."; tmp = path2.Dir(tmp) {
		if repo, found := b.cache.Load(tmp); found {
			log.Println("[CACHE]", path)
			return repo.(*git.Repository), tmp, nil
		}
	}

	var base string
	for tmp := path; tmp != "."; tmp = path2.Dir(tmp) {
		url := mapUrl(tmp)
		log.Println("[MAPPED]", tmp, "->", url)
		if _, err := ListRemote(url); err != nil {
			continue
		}
		base = tmp
		break
	}

	if base != "" {
		url := mapUrl(base)
		cachePath := filepath.Join("c:\\temp", "cache", uuid.New().String())
		repo, err := git.PlainClone(cachePath, true, &git.CloneOptions{
			URL:        url,
			RemoteName: git.DefaultRemoteName,
		})
		if err != nil {
			return nil, "", errors.Wrapf(err, "failed to clone repository '%s'", base)
		}
		log.Println("[CLONE]", base, "->", cachePath)
		b.cache.Store(base, repo)
		return repo, base, nil
	}

	log.Println("[MISSED]", path)
	return nil, "", ErrNotFound
}

func (b *Direct) GetList(path, major string) ([]string, error) {
	for tmp := path; tmp != "."; tmp = path2.Dir(tmp) {
		refs, err := ListRemote(tmp)
		if err != nil {
			continue
		}

		versions := map[string]bool{}
		submodule := splitSubModule(path, tmp)
		for _, ref := range refs {
			if !ref.Name().IsTag() {
				continue
			}

			tag := ref.Name().Short()
			if !strings.HasPrefix(tag, submodule) {
				continue
			}

			version := strings.TrimPrefix(strings.TrimPrefix(tag, submodule), "/")
			if err := module.CheckPathMajor(version, major); err != nil {
				continue
			}

			version = module.CanonicalVersion(version)
			versions[version] = true
		}

		if len(versions) == 0 {
			return nil, ErrNotFound
		}

		return maps.Keys(versions), nil
	}

	return nil, ErrNotFound
}

func (b *Direct) GetLatest(path, major string) (string, time.Time, error) {
	repo, base, err := b.getRepository(path)
	if err != nil {
		return "", time.Unix(0, 0), err
	}

	submodule := splitSubModule(path, base)

	tags, err := repo.Tags()
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to get tags")
	}
	defer tags.Close()

	version := ""
	var latest *object.Commit
	for {
		tag, err := tags.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if !strings.HasPrefix(tag.Name().Short(), submodule) {
			continue
		}

		tmp := strings.TrimPrefix(strings.TrimPrefix(tag.Name().Short(), submodule), "/")
		if err := module.CheckPathMajor(tmp, major); err != nil {
			continue
		}

		if commit, err := repo.CommitObject(tag.Hash()); err != nil {
			log.Println("failed to lookup commit:", err)
			continue
		} else if latest == nil {
			latest = commit
			version = tmp
		} else if latest.Committer.When.Before(commit.Committer.When) {
			latest = commit
			version = tmp
		}
	}

	if latest != nil {
		return version, latest.Committer.When.UTC(), nil
	} else if submodule == "" {
		ref, err := repo.Head()
		if err != nil {
			return "", time.Unix(0, 0), ErrNotFound
		}

		commit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			return "", time.Unix(0, 0), ErrNotFound
		}
		version = module.PseudoVersion("v0.0.0", "", commit.Committer.When.UTC(), ref.Hash().String())
		return version, commit.Committer.When.UTC(), nil
	}

	return "", time.Unix(0, 0), ErrNotFound
}

func (b *Direct) GetModule(path, version string) (string, error) {
	path, major, _ := module.SplitPathVersion(path)
	repo, base, err := b.getRepository(path)
	if err != nil {
		return "", err
	}
	submodule := splitSubModule(path, base)

	rev := version
	if module.IsPseudoVersion(version) {
		rev, err = module.PseudoVersionRev(version)
		if err != nil {
			return "", errors.Wrap(err, "failed to get revision from pseudo version")
		}
	} else if submodule != "" {
		rev = submodule + "/" + version
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return "", ErrNotFound
		}
		return "", errors.Wrap(err, "failed to resolve revision")
	}

	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return "", errors.Wrap(err, "failed to get commit object")
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", errors.Wrap(err, "failed to get tree for commit")
	}

	modFilePath := "go.mod"
	if submodule != "" {
		modFilePath = submodule + "/" + modFilePath
	}

	// tree.File does not work on folders, we need to use Tree
	tmpModFilePath := strings.TrimPrefix(major+"/"+modFilePath, "/")
	if _, err := tree.File(tmpModFilePath); err == nil {
		modFilePath = tmpModFilePath
	}

	f, err := tree.File(modFilePath)
	if err != nil {
		if err == object.ErrFileNotFound && major == "" && submodule == "" {
			return "module " + path + "\n", nil
		}
		return "", ErrNotFound
	}

	return f.Contents()
}

func (b *Direct) GetInfo(path, version string) (string, time.Time, error) {
	path, _, _ = module.SplitPathVersion(path)
	repo, base, err := b.getRepository(path)
	if err != nil {
		return "", time.Unix(0, 0), err
	}
	submodule := splitSubModule(path, base)

	rev := version
	if module.IsPseudoVersion(version) {
		rev, _ = module.PseudoVersionRev(version)
	} else if submodule != "" {
		rev = submodule + "/" + version
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to resolve revision")
	}

	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to get commit object")
	}

	return version, commit.Committer.When.UTC(), nil
}

// GetArchive returns a zip file of the modules contents at a certain version
// ignore any nested modules (they will be requested independently)
// ignore vendor directory
// add the root LICENSE from base directory if submodule does not include one
func (b *Direct) GetArchive(path, version string) (io.Reader, error) {
	prefix, major, _ := module.SplitPathVersion(path)
	repo, base, err := b.getRepository(prefix)
	if err != nil {
		return nil, err
	}
	submodule := splitSubModule(prefix, base)

	rev := version
	if module.IsPseudoVersion(version) {
		rev, err = module.PseudoVersionRev(version)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get revision from pseudo version")
		}
	} else if submodule != "" {
		rev = submodule + "/" + version
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, ErrNotFound
		}
		return nil, errors.Wrap(err, "failed to resolve revision")
	}

	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get commit object")
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tree for commit")
	}

	// we get the subtree (ie subdirectory) that contains the module
	moduleTree := tree
	if submodule != "" {
		if tmp, err := tree.Tree(strings.TrimPrefix(submodule, "/")); err == nil {
			moduleTree = tmp
		}
	}
	if major != "" {
		if tmp, err := tree.Tree(strings.TrimPrefix(major, "/")); err == nil {
			moduleTree = tmp
		}
	}

	var buffer bytes.Buffer
	z := zip.NewWriter(&buffer)
	defer z.Close()

	// Use the LICENSE file in the root directory if one does not exist in the module's directory
	if _, err := moduleTree.File("LICENSE"); err != nil {
		// if this exists, it will get copied later
		if f, err := tree.File("LICENSE"); err == nil {
			filename := strings.TrimPrefix(f.Name, submodule)
			filename = path2.Join(path+"@"+version, filename)
			if fr, err := f.Reader(); err != nil {
				return nil, errors.Wrap(err, "failed to open license file in repo")
			} else if fw, err := z.Create(filename); err != nil {
				fr.Close()
				return nil, errors.Wrap(err, "failed to create license file inside zip")
			} else {
				if _, err := io.Copy(fw, fr); err != nil {
					fr.Close()
					return nil, errors.Wrap(err, "failed to copy license to zip file")
				}
				fr.Close()
			}
		}
	}

	// Build a list of nested modules that need to be ignored
	var ignore []string
	if err := moduleTree.Files().ForEach(func(f *object.File) error {
		if submodule != "" {
			if f.Name == submodule+"/go.mod" {
				return nil
			}
		} else if f.Name == "go.mod" {
			return nil
		}

		if strings.HasSuffix(f.Name, "go.mod") {
			ignore = append(ignore, path2.Dir(f.Name))
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to build a list of folders to ignore")
	}

	if err := moduleTree.Files().ForEach(func(f *object.File) error {
		if f.Mode == filemode.Symlink {
			// Symlinks are excluded
			return nil
		}

		//ignore any directories that have their own modules
		for _, tmp := range ignore {
			if strings.HasPrefix(f.Name, tmp) {
				//log.Println("ignored:", f.Name)
				return nil
			}
		}

		filename := strings.TrimPrefix(f.Name, submodule)
		filename = path2.Join(path+"@"+version, filename)
		fw, err := z.Create(filename)
		if err != nil {
			return err
		}

		fr, err := f.Reader()
		if err != nil {
			return err
		}

		_, err = io.Copy(fw, fr)
		if err != nil {
			return err
		}

		//log.Println("added:", f.Name, "->", filename)
		return fr.Close()
	}); err != nil {
		return nil, err
	}

	return &buffer, nil
}
