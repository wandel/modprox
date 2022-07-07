package backend

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
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
	"github.com/pkg/errors"
	"github.com/wandel/modprox/utils"

	"golang.org/x/exp/maps"
	"golang.org/x/mod/module"
)

type Git struct {
	cache    sync.Map
	CacheDir string
}

func (d *Git) Load() error {
	if d.CacheDir == "" {
		return errors.New("Cache Directory is required")
	}

	if err := filepath.Walk(d.CacheDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		} else if !info.IsDir() {
			log.Println("failed to open git repository:", path, err)
			return filepath.SkipDir
		} else if repo, err := git.PlainOpen(path); err != nil {
			return nil
		} else {

			// work around for short hashes not resolving on first request for repos opened with PlainOpen
			// https://github.com/go-git/go-git/issues/148
			repo.ResolveRevision(plumbing.Revision("HEAD"))

			key, err := filepath.Rel(d.CacheDir, path)
			if err != nil {
				log.Println("failed to load cache", err)
				return filepath.SkipDir
			}
			key = filepath.ToSlash(key)
			//log.Println("[CACHE]", path, "->", key)
			d.cache.Store(key, &Repo{r: repo})
			return filepath.SkipDir
		}
	}); err != nil {
		return errors.Wrap(err, "failed to load repositories from cache")
	}

	log.Println("loaded cache")
	return nil
}

type Repo struct {
	sync.Mutex
	r *git.Repository
}

type Tag struct {
	Name string
	Date time.Time
}

func (r *Repo) Tags() ([]Tag, error) {
	if r.r == nil {
		return nil, errors.New("repository was not initialized")
	}

	var tags []Tag
	refs, err := r.r.Tags()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list tags")
	}
	defer refs.Close()

	if err := refs.ForEach(func(ref *plumbing.Reference) error {
		switch obj, err := r.r.TagObject(ref.Hash()); err {
		case nil:
			commit, err := obj.Commit()
			if err != nil {
				return errors.Wrapf(err, "failed to get commit for tag '%s'", obj.Name)
			}
			tags = append(tags, Tag{obj.Name, commit.Committer.When.UTC()})
		case plumbing.ErrObjectNotFound:
			commit, err := r.r.CommitObject(ref.Hash())
			if err != nil {
				return errors.Wrapf(err, "failed to get commit for tag: '%s'", ref.String())
			}
			tags = append(tags, Tag{ref.Name().Short(), commit.Committer.When.UTC()})
		default:
			return errors.Wrapf(err, "failed to resolve TagObject for %s", ref.String())
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to list tags")
	}

	return tags, nil
}

func splitSubModule(path, base string) string {
	return strings.TrimPrefix(strings.TrimPrefix(path, base), "/")
}

func ListRemote(path string) ([]*plumbing.Reference, error) {
	url, _, err := utils.MapPath(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to map to a repository")
	}

	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{
			fmt.Sprintf("https://%s", url),
		},
	})

	refs, err := remote.List(&git.ListOptions{})
	if errors.Is(err, transport.ErrRepositoryNotFound) || errors.Is(err, transport.ErrAuthenticationRequired) {
		log.Println("[MISSING]", path)
		return nil, ErrNotFound
	}
	return refs, err
}

func (b *Git) getRepository(path string) (*Repo, string, error) {
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
			return repo.(*Repo), tmp, nil
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

	if base == "" {
		log.Println("[MISSING]", path)
		return nil, "", ErrNotFound
	}

	placeholder := &Repo{}
	placeholder.Lock()
	defer placeholder.Unlock()

	if r2, loaded := b.cache.LoadOrStore(base, placeholder); loaded {
		// another request has / is loading this repo
		return r2.(*Repo), base, nil
	}

	url := mapUrl(base)
	cachePath := filepath.Join(b.CacheDir, base)
	repo, err := git.PlainClone(cachePath, true, &git.CloneOptions{
		URL:        fmt.Sprintf("https://%s", url),
		RemoteName: git.DefaultRemoteName,
		Tags:       git.AllTags,
	})
	if err != nil {
		log.Printf("[ERROR] failed to clone repository: url=%s, base=%s, err=%s\n", url, base, err)
		return nil, "", errors.Wrapf(err, "failed to clone repository '%s'", base)
	}

	log.Println("[CLONE]", base, "->", cachePath)
	placeholder.r = repo
	return placeholder, base, nil
}

func (b *Git) GetList(path, major string) ([]string, error) {
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

		return maps.Keys(versions), nil
	}

	return nil, ErrNotFound
}

func (b *Git) GetLatest(path, major string) (string, time.Time, error) {
	repo, base, err := b.getRepository(path)
	if err != nil {
		return "", time.Unix(0, 0), err
	}

	repo.Lock()
	defer repo.Unlock()
	if repo.r == nil {
		return "", time.Unix(0, 0).UTC(), ErrNotFound
	}

	if err := repo.r.Fetch(&git.FetchOptions{
		Force: true,
		Tags:  git.AllTags,
	}); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			log.Println("failed to fetch updates:", err)
		}
	} else {
		log.Println("[UPDATED]", path)
	}

	submodule := splitSubModule(path, base)
	var latest *plumbing.Reference
	var commit *object.Commit

	tags, err := repo.r.Tags()
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to get tags")
	}

	if err := tags.ForEach(func(ref *plumbing.Reference) error {
		if !strings.HasPrefix(ref.Name().Short(), submodule) {
			return nil
		}

		tmp := strings.TrimPrefix(strings.TrimPrefix(ref.Name().Short(), submodule), "/")
		if err := module.CheckPathMajor(tmp, major); err != nil {
			return nil
		}

		switch obj, err := repo.r.TagObject(ref.Hash()); err {
		case nil:
			tmp, err := obj.Commit()
			if err != nil {
				return errors.Wrapf(err, "failed to get commit for tag '%s'", obj.Name)
			} else if commit == nil || commit.Committer.When.Before(tmp.Committer.When) {
				latest = ref
				commit = tmp
			}
		case plumbing.ErrObjectNotFound:
			tmp, err := repo.r.CommitObject(ref.Hash())
			if err != nil {
				return errors.Wrapf(err, "failed to get commit for tag: '%s'", ref.String())
			} else if commit == nil || commit.Committer.When.Before(tmp.Committer.When) {
				latest = ref
				commit = tmp
			}
		default:
			return errors.Wrapf(err, "failed to resolve TagObject for %s", ref.String())
		}

		return nil
	}); err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to list tags")
	}

	if latest != nil {
		version := strings.TrimPrefix(strings.TrimPrefix(latest.Name().Short(), submodule), "/")
		return version, commit.Committer.When.UTC(), nil
	} else if submodule == "" {
		ref, err := repo.r.Head()
		if err != nil {
			return "", time.Unix(0, 0), ErrNotFound
		}

		commit, err := repo.r.CommitObject(ref.Hash())
		if err != nil {
			return "", time.Unix(0, 0), ErrNotFound
		}

		version := module.PseudoVersion("v0", "", commit.Committer.When.UTC(), commit.Hash.String()[:12])
		return version, commit.Committer.When.UTC(), nil
	}

	return "", time.Unix(0, 0), ErrNotFound
}

func (b *Git) GetModule(path, version string) (string, error) {
	incompatible := strings.HasSuffix(version, "+incompatible")
	version = strings.TrimSuffix(version, "+incompatible")
	path, major, _ := module.SplitPathVersion(path)
	if incompatible && major != "" {
		return "", errors.Errorf("major version suffix should not be provided on a +incompatible version: version='%s', path='%s'", path, version)
	}
	repo, base, err := b.getRepository(path)
	if err != nil {
		return "", err
	}
	submodule := splitSubModule(path, base)

	repo.Lock()
	defer repo.Unlock()
	if repo.r == nil {
		return "", errors.Wrap(err, "repository was nil")
	}

	if err := repo.r.Fetch(&git.FetchOptions{
		RemoteName: git.DefaultRemoteName,
	}); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			log.Println("failed to fetch updates:", err)
		}
	} else {
		log.Println("[UPDATED]", path)
	}

	rev := version
	if module.IsPseudoVersion(version) {
		rev, err = module.PseudoVersionRev(version)
		if err != nil {
			return "", errors.Wrap(err, "failed to get revision from pseudo version")
		}
	} else if submodule != "" {
		rev = submodule + "/" + version
	}

	hash, err := repo.r.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			log.Println("[MOD] no reference found for revision", rev)
			return "", ErrOutOfDate
		}
		return "", errors.Wrap(err, "failed to resolve revision")
	}

	commit, err := repo.r.CommitObject(*hash)
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
		if err == object.ErrFileNotFound && submodule == "" {
			if major == "" {
				// generate a fake go.mod for v0/v1 for normal packages
				return "module " + path + "\n", nil
			} else if strings.HasPrefix(major, ".") {
				// we generate a fake go.mod for any gopkg.in
				return "module " + path + major + "\n", nil
			} else {
				return "", errors.New("invalid module.... missing go.mod file")
			}
		}
		return "", ErrNotFound
	}

	return f.Contents()
}

func (b *Git) GetInfo(path, version string) (string, time.Time, error) {
	incompatible := strings.HasSuffix(version, "+incompatible")
	version = strings.TrimSuffix(version, "+incompatible")
	path, _, _ = module.SplitPathVersion(path)
	repo, base, err := b.getRepository(path)
	if err != nil {
		return "", time.Unix(0, 0), err
	}
	submodule := splitSubModule(path, base)

	repo.Lock()
	defer repo.Unlock()
	if repo.r == nil {
		return "", time.Unix(0, 0), ErrNotFound
	}

	if err := repo.r.Fetch(&git.FetchOptions{
		RemoteName: git.DefaultRemoteName,
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/heads/*:refs/remotes/origin/heads/*"),
		},
	}); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			log.Println("failed to fetch updates:", err)
		}
	} else {
		log.Println("[UPDATED]", path)
	}

	rev := version
	if module.IsPseudoVersion(version) {
		rev, _ = module.PseudoVersionRev(version)
	} else if submodule != "" {
		rev = submodule + "/" + version
	}

	hash, err := repo.r.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return "", time.Unix(0, 0), ErrOutOfDate
		}
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to resolve revision")
	}

	commit, err := repo.r.CommitObject(*hash)
	if err != nil {
		return "", time.Unix(0, 0), errors.Wrap(err, "failed to get commit object")
	}

	if incompatible {
		version = version + "+incompatible"
	}
	return version, commit.Committer.When.UTC(), nil
}

// GetArchive returns a zip file of the modules contents at a certain version
// ignore any nested modules (they will be requested independently)
// ignore vendor directory
// add the root LICENSE from base directory if submodule does not include one
func (b *Git) GetArchive(path, version string) (io.Reader, error) {
	prefix, major, _ := module.SplitPathVersion(path)
	repo, base, err := b.getRepository(prefix)
	if err != nil {
		return nil, err
	}

	repo.Lock()
	defer repo.Unlock()
	if repo.r == nil {
		return nil, ErrNotFound
	}

	if err := repo.r.Fetch(&git.FetchOptions{
		RemoteName: git.DefaultRemoteName,
	}); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			log.Println("failed to fetch updates:", err)
		}
	} else {
		log.Println("[UPDATED]", path)
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

	hash, err := repo.r.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, ErrOutOfDate
		}
		return nil, errors.Wrap(err, "failed to resolve revision")
	}

	commit, err := repo.r.CommitObject(*hash)
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

	// TODO: use golang.org/x/mod/zip.Create() so i dont have to maintain this

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
		if strings.HasSuffix(f.Name, "/go.mod") {
			ignore = append(ignore, path2.Dir(f.Name)+"/")
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to build a list of folders to ignore")
	}

	log.Println("[IGNORE]", ignore)
	if err := moduleTree.Files().ForEach(func(f *object.File) error {
		if f.Mode == filemode.Symlink {
			// Symlinks are excluded
			return nil
		}

		// ignore any vendor directories
		if strings.HasPrefix(f.Name, "vendor/") || strings.Contains(f.Name, "/vendor/") {
			if strings.HasSuffix(f.Name, "vendor/modules.txt") {
				// vendor/modules.txt is allowed
			} else if strings.HasSuffix(f.Name, "vendor/vendor.json") {
				// vendor/vendor.json is allowed ?
			} else {
				return nil
			}
		}

		//ignore any directories that have their own modules
		for _, tmp := range ignore {
			if strings.HasPrefix(f.Name, tmp) {
				return nil
			}
		}

		filename := path2.Join(path+"@"+version, f.Name)
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
