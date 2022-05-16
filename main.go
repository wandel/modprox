package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	app := cli.NewApp()
	app.Name = "ModProx"
	app.Usage = "Golang Module Proxy"

	app.Action = ListenAction

	if err := app.Run(os.Args); err != nil {
		log.Fatalln("failed to run app:", err)
	}
}

// ListHandler provides a list of valid versions (git tags) for a module
func ListHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// probably can do a better job of generating the git module path
	url := fmt.Sprintf("https://%s.git", vars["module"])
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "remote",
		URLs: []string{url},
	})

	if refs, err := remote.List(&git.ListOptions{}); err != nil {
		if err == transport.ErrRepositoryNotFound {
			// url is not a valid repo path, we mimic github.com and return status 410
			w.WriteHeader(http.StatusGone)
		} else {
			// something else went wrong :(
			w.WriteHeader(http.StatusInternalServerError)
			// we probably don't need to be logging details from the error to the user.
			fmt.Fprintln(w, "failed to list remote branches/tags", err)
		}
	} else {
		for _, ref := range refs {
			name := ref.Name()
			short := name.Short()
			if !name.IsTag() {
				// only tags are relevant
				continue
			}

			if vars["version"] != "" {
				// if version is specified, we will filter based on that
				if strings.HasPrefix(short, vars["version"]) {
					//is a valid tag, so we supply it to the client
					fmt.Fprintln(w, short)
				}
			} else if !strings.HasPrefix(short, "v0") && !strings.HasPrefix(short, "v1") {
				// if version is not specified, we will accept either v0 or v1 tags by default
				fmt.Fprintln(w, short)
			}
		}
	}
}

func ModHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.mod", vars["module"], vars["version"])
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(w, "failed to get mod: %s", err)
		log.Println("failed to get mod:", err)
		w.WriteHeader(http.StatusBadGateway)
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	log.Printf("%s (%d - %s)\n", r.URL.String(), resp.StatusCode, resp.Status)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Println("failed to write data:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func InfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.info", vars["module"], vars["version"])
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(w, "failed to get info %s", err)
		log.Println("failed to get info:", err)
		w.WriteHeader(http.StatusBadGateway)
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	log.Printf("%s (%d - %s)\n", r.URL.String(), resp.StatusCode, resp.Status)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Println("failed to write data:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func ArchiveHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.zip", vars["module"], vars["version"])
	revision := plumbing.NewTagReferenceName(vars["version"])
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:             url,
		ReferenceName:   plumbing.ReferenceName(revision),
		SingleBranch:    true,
		Depth:           1,
		Progress:        os.Stdout,
		InsecureSkipTLS: false,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "failed to shallow clone repository", err)
	}

	// if we write directly to the ResponseWriter we are forced to return a 200 - OK if anything goes wrong.
	// So instead we write to a temporary inmemory buffer, then once the zip has been completed successfully, we write
	// it out to the response.
	buffer := new(bytes.Buffer)
	if err := WriteZip(repo, buffer, revision.Short()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "failed to write zip file", err)
	}

	io.Copy(w, buffer)
}

func ListenAction(ctx *cli.Context) error {
	router := mux.NewRouter()
	router.HandleFunc("/{module:.+}/{version:v[1-9]+}/@v/list", ListHandler)
	router.HandleFunc("/{module:.+}/@v/list", ListHandler)
	router.HandleFunc("/{module:.+}/@v/{version}.mod", ModHandler)
	router.HandleFunc("/{module:.+}/@v/{version}.info", InfoHandler)
	router.HandleFunc("/{module:.+}/@v/{version}.zip", ArchiveHandler)
	http.Handle("/", router)

	if err := http.ListenAndServe(":8000", nil); err != nil {
		return err
	}

	return nil
}

func ListTags(url, version string) ([]string, error) {
	log.Println("setting up new remote")
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "remote",
		URLs: []string{url},
	})

	log.Println("listing remote stuff")
	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		if err == transport.ErrRepositoryNotFound {
			return nil, err
		} else {
			return nil, errors.Wrap(err, "failed to list remote branches/tags")
		}
	}

	log.Println("stuff:")
	var tags []string
	for _, ref := range refs {
		name := ref.Name()
		if name.IsTag() && strings.HasPrefix(name.Short(), version) {
			tags = append(tags, name.Short())
		}
	}
	return tags, nil
}

func BuildArchive(url, tag string, w io.Writer) error {

	return nil
}

// WriteZip will stream the content of the repository to w as a zip file.
// revision can be anything supported by ResolveRevision(), but please
// note that short hashes are not supported for repositories opened using
// PlainOpen(). See: https://github.com/go-git/go-git/issues/148
func WriteZip(repo *git.Repository, w io.Writer, revision string) error {
	hash, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return err
	}

	// Get the corresponding commit hash.
	obj, err := repo.CommitObject(*hash)
	if err != nil {
		return err
	}

	// Let's have a look at the tree at that commit.
	tree, err := repo.TreeObject(obj.TreeHash)
	if err != nil {
		return err
	}

	z := zip.NewWriter(w)

	addFile := func(f *object.File) error {
		log.Println("added:", f.Name)
		fw, err := z.Create(f.Name)
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

		return fr.Close()
	}

	err = tree.Files().ForEach(addFile)
	if err != nil {
		return err
	}

	return z.Close()
}

func StartAction(ctx *cli.Context) error {
	url := "https://github.com/llvm/llvm-project/llvm"
	tag := "llvmorg-14.0.3"

	//_, filename := path.Split(url)
	//filename = fmt.Sprintf("%s@%s.zip", filename, tag)
	//f, err := os.Create(filename)
	//if err != nil {
	//	return errors.Wrapf(err, "failed to create file %s", filename)
	//}
	//defer f.Close()
	//
	//if err := BuildArchive(url, tag, f); err != nil {
	//	os.Remove(filename)
	//	return errors.Wrap(err, "failed to build archive")
	//}

	tag = "llvmorg"
	tags, err := ListTags(url, tag)
	if err != nil {
		return errors.Wrap(err, "failed to list versions")
	}

	for i, tag := range tags {
		log.Println(i, tag)
	}

	//key, err := ssh.NewPublicKeysFromFile("git", "c:\\users\\brett\\.ssh\\id_ed25519", "")
	//if err != nil {
	//	return errors.Wrap(err, "failed to open private key")
	//}
	//
	//log.Println("Cloning base repository")
	//repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
	//	URL:      "git@github.com:wandel/pdbgen.git",
	//	Auth:     key,
	//	Progress: os.Stdout,
	//})
	//if err != nil {
	//	return errors.Wrap(err, "failed to clone repository")
	//}

	//log.Println("creating remote")
	//remote, err := repo.CreateRemote(&config.RemoteConfig{
	//	Name: "backup",
	//	URLs: []string{"git@gitlab.com:bwandel/pdbgen.git"},
	//})
	//if err != nil {
	//	return errors.Wrap(err, "failed to create remote")
	//}
	//
	//log.Println("pushing to backup")
	//if err := remote.Push(&git.PushOptions{
	//	RemoteName: "backup",
	//	Auth:       key,
	//	Progress:   os.Stdout,
	//}); err != nil {
	//	return errors.Wrap(err, "failed to push to remote")
	//}

	return nil
}
