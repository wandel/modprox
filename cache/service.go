package cache

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	"golang.org/x/mod/module"
	path2 "path"
	"strings"
	"sync"
)

type CacheService struct {
	lock   sync.RWMutex
	lookup map[string]string
	cache  sync.Map
}

// LookupRepository searches the local cache for
func (cs *CacheService) LookupRepository(path string) (*git.Repository, string) {
	for tmp := path; tmp != "."; tmp = path2.Dir(tmp) {
		if repo, found := cs.cache.Load(path); found {
			return repo.(*git.Repository), tmp
		}
	}

	return nil, ""
}

func (cs *CacheService) FetchRepository(path string) (*git.Repository, string, error) {
	repo, base := cs.LookupRepository(path)
	if repo != nil {
		return repo, base, nil
	}

	for tmp := path; tmp != "."; tmp = path2.Dir(tmp) {
		remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
			Name: "origin",
			URLs: []string{
				fmt.Sprintf("https://%s.git", tmp),
			},
		})

		if _, err := remote.List(&git.ListOptions{}); err != nil {
			continue
		}

	}

	return nil, "", transport.ErrRepositoryNotFound
}

func (cs *CacheService) GetTags(path string) ([]string, error) {
	for tmp := path; path != "."; tmp = path2.Dir(path) {
		remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
			Name: "origin",
			URLs: []string{
				fmt.Sprintf("https://%s.git", tmp),
			},
		})

		var versions map[string]bool
		subpath := strings.TrimPrefix(strings.TrimPrefix(path, tmp), "/")
		refs, err := remote.List(&git.ListOptions{})
		if err != nil {
			continue
		}

		for _, ref := range refs {
			if !ref.Name().IsTag() {
				continue
			}

			tag := ref.Name().Short()
			if !strings.HasPrefix(tag, subpath) {
				continue
			}

			version := module.CanonicalVersion(strings.TrimPrefix(tag, subpath))
			versions[version] = true
		}

		return maps.Keys(versions), nil
	}

	return nil, errors.Errorf("no repositories found for '%s'", path)
}

func (cs *CacheService) GetModFile(path, version string) (string, error) {
	return "", errors.New("not implemented yet")
}

func (cs *CacheService) GetArchive(path, version string) ([]byte, error) {
	return nil, errors.New("not implemented yet")
}

//func (b Direct) buildURL(base string) string {
//	return fmt.Sprintf("https://%s.git", base)
//}
//
//func (b Direct) RemoteList(base string) ([]*plumbing.Reference, error) {
//	url := b.buildURL(base)
//	if refs, err := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
//		URLs: []string{url},
//	}).List(&git.ListOptions{}); err != nil {
//		if errors.Is(err, transport.ErrRepositoryNotFound) {
//			return nil, ErrNotFound
//		} else if errors.Is(err, transport.ErrAuthenticationRequired) {
//			return nil, ErrNotFound
//		}
//		return nil, errors.Wrapf(err, "failed to ls-remote '%s'", url)
//	} else {
//		return refs, nil
//	}
//}
//
//func (b Direct) LookupRepoPath(path string) (string, error) {
//	base := path
//	for {
//		subpath := strings.TrimPrefix(path, base)
//		if _, err := b.RemoteList(base, subpath); err == nil {
//			return base, nil
//		} else if !errors.Is(err, ErrNotFound) {
//			return "", err
//		}
//
//		base = path2.Dir(base)
//		if base == "." {
//			return "", ErrNotFound
//		}
//	}
//
//	return "", nil
//}
