package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/wandel/modprox/backend"
	"golang.org/x/mod/module"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var source backend.Backend

func main() {
	app := cli.NewApp()
	app.Name = "ModProx"
	app.Usage = "Golang Module Proxy"
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "username", Value: "git", Usage: "the username to use"},
		&cli.StringFlag{Name: "password", Value: "", Usage: "the password to use"},
		&cli.StringFlag{Name: "privatekey", Value: "", Usage: "the ssh private key to use"},
		cli.StringFlag{Name: "token", Value: "", Usage: "gitlab personal access token"},
	}

	app.Before = func(ctx *cli.Context) error {
		token := ctx.GlobalString("token")
		gitlab, err := backend.NewGitLab(token, "mirror8")
		if err != nil {
			return errors.Wrap(err, "failed to create gitlab backend")
		}

		source = backend.NewMultiBackend(gitlab)
		return nil
	}

	app.Action = func(ctx *cli.Context) error {
		router := mux.NewRouter()
		router.HandleFunc("/{module:.+}/@v/list", ListHandler)
		router.HandleFunc("/{module:.+}/@latest", LatestHandler)
		router.HandleFunc("/{module:.+}/@v/{version}.mod", ModHandler)
		router.HandleFunc("/{module:.+}/@v/{version}.info", InfoHandler)
		router.HandleFunc("/{module:.+}/@v/{version}.zip", ArchiveHandler)

		log.Println("listening on 127.0.0.1:8000")
		if err := http.ListenAndServe("127.0.0.1:8000", router); err != nil {
			return err
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln("failed to run app:", err)
	}
}

//func getAuth(ctx *cli.Context) (transport.AuthMethod, error) {
//	username := ctx.GlobalString("username")
//	password := ctx.GlobalString("password")
//	privatekey := ctx.GlobalString("privatekey")
//
//	if "privatekey" != "" {
//		auth, err := ssh.NewPublicKeysFromFile(username, privatekey, password)
//		if err != nil {
//			return nil, errors.Wrapf(err, "failed to load ssh key from '%s'", privatekey)
//		}
//		return auth, nil
//	} else if password != "" {
//		return &http2.BasicAuth{Username: username, Password: password}, nil
//	}
//
//	return nil, errors.New("no auth method  provided")
//}
//
//func isStale(src, dst []*plumbing.Reference) bool {
//	for _, x := range src {
//		if strings.HasPrefix(x.Name().String(), "refs/pull") {
//			continue
//		}
//		var found bool
//		for _, y := range dst {
//			log.Println(x.Name(), "-", y.Name())
//			if x.Hash() == y.Hash() {
//				found = true
//				break
//			}
//		}
//		if !found {
//			return true
//		}
//	}
//
//	for _, x := range dst {
//		var found bool
//		for _, y := range src {
//			log.Println(x.Name(), "=", y.Name())
//			if x.Hash() == y.Hash() {
//				found = true
//				break
//			}
//		}
//		if !found {
//			return true
//		}
//	}
//
//	return false
//}
//
//func SyncAction(ctx *cli.Context) error {
//	srcUrl := ctx.String("source")
//	dstUrl := ctx.String("destination")
//
//	auth, err := getAuth(ctx)
//	if err != nil {
//		return errors.Wrap(err, "failed to get authentication details")
//	}
//
//	log.Printf("syncing '%s' to '%s'\n", srcUrl, dstUrl)
//	src := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
//		URLs: []string{srcUrl},
//	})
//
//	dst := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
//		URLs: []string{dstUrl},
//	})
//
//	srcRefs, err := src.List(&git.ListOptions{})
//	if err != nil {
//		return errors.Wrap(err, "failed to list references on the source")
//	}
//
//	log.Println("source:")
//	for _, ref := range srcRefs {
//		if strings.HasPrefix(ref.Name().String(), "refs/pull") {
//			continue
//		}
//		log.Println(ref.Name())
//	}
//
//	dstRefs, err := dst.List(&git.ListOptions{Auth: auth})
//	if err != nil && err != transport.ErrEmptyRemoteRepository {
//		return errors.Wrap(err, "failed to list references on the destination")
//	}
//
//	log.Println("destination:")
//	for _, ref := range dstRefs {
//		log.Println(ref.Name())
//	}
//
//	if !isStale(srcRefs, dstRefs) {
//		log.Println("destination is up to date")
//		return nil
//	}
//
//	log.Println("cloning", srcUrl)
//	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
//		URL:      srcUrl,
//		Progress: os.Stdout,
//	})
//	if err != nil {
//		return errors.Wrap(err, "failed to clone source repository")
//	}
//
//	if _, err := repo.CreateRemote(&config.RemoteConfig{
//		Name: "destination",
//		URLs: []string{dstUrl},
//	}); err != nil {
//		return errors.Wrap(err, "failed to create remote")
//	}
//
//	log.Println("pushing to", dstUrl)
//	if err := repo.Push(&git.PushOptions{
//		RemoteName: "destination",
//		Progress:   os.Stdout,
//		Auth:       auth,
//		Force:      true,
//		Prune:      true,
//		RefSpecs: []config.RefSpec{
//			"+refs/heads/*:refs/heads/*",
//			"+refs/tags/*:refs/tags/*",
//		},
//	}); err != nil {
//		return errors.Wrap(err, "failed to push repo to destination")
//	}
//
//	return nil
//}

func MapPath(path string) (string, string, error) {
	// not sure if this is required...
	mappings := map[string]string{
		"google.golang.org/grpc":         "github.com/grpc/grpc-go",
		"google.golang.org/protobuf":     "github.com/protocolbuffers/protobuf-go",
		"google.golang.org/api":          "github.com/googleapis/google-api-go-client",
		"google.golang.org/genproto":     "github.com/googleapis/go-genproto",
		"go.opentelemetry.io/proto/otlp": "github.com/open-telemetry/opentelemetry-proto-go",
		"google.golang.org/appengine":    "github.com/golang/appengine",
		"honnef.co/go/tools":             "github.com/dominikh/go-tools",
	}

	if err := module.CheckPath(path); err != nil {
		return "", "", errors.Wrap(err, "invalid path")
	}

	path, major, _ := module.SplitPathVersion(path)
	path = strings.TrimPrefix(path, "/")
	major = strings.TrimPrefix(major, ".")
	major = strings.TrimPrefix(major, "/")

	if strings.HasPrefix(path, "gopkg.in/") {
		switch parts := strings.Split(path, "/"); len(parts) {
		case 2:
			path = parts[0] + "/" + "go-" + parts[1] + "/" + parts[1]
		case 3:
			break
		default:
			return "", "", errors.Errorf("invalid gopkg.in path '%s': unexpected number of '/'", path)
		}
		path = strings.Replace(path, "gopkg.in/", "github.com/", 1)
	} else if strings.HasPrefix(path, "golang.org/x") {
		path = strings.Replace(path, "golang.org/x", "github.com/golang", 1)
	} else if strings.HasPrefix(path, "cloud.google.com/go") {
		path = strings.Replace(path, "cloud.google.com/go", "github.com/googleapis/google-cloud-go", 1)
	} else if value, ok := mappings[path]; ok {
		path = value
	}

	if major == "v1" {
		major = ""
	}

	return path, major, nil
}

// ListHandler provides a list of valid versions (git tags) for a module
func ListHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("===", r.URL, "===")
	unmapped := mux.Vars(r)["module"]
	mapped, major, err := MapPath(unmapped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	path, err := module.UnescapePath(mapped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if versions, err := source.GetList(path, major); err != nil {
		if errors.Is(err, backend.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		for _, version := range versions {
			fmt.Fprintln(w, version)
		}
	}
}

func LatestHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("===", r.URL, "===")
	unmapped := mux.Vars(r)["module"]
	mapped, major, err := MapPath(unmapped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	path, err := module.UnescapePath(mapped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	latest, timestamp, err := source.GetLatest(path, major)
	if err != nil {
		if errors.Is(err, backend.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Version string
		Time    string
	}{
		Version: latest,
		Time:    timestamp.UTC().Format(time.RFC3339),
	})
}

func ModHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("===", r.URL, "===")
	unmapped, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mapped, major, err := MapPath(unmapped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if major != "" {
		mapped = mapped + "/" + major
	}

	version, err := module.UnescapeVersion(mux.Vars(r)["version"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mod, err := source.GetModule(mapped, version)
	if err != nil {
		if errors.Is(err, backend.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if strings.Contains(mod, mapped) {
		mod = strings.Replace(mod, mapped, unmapped, 1)
	}

	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	if _, err := io.WriteString(w, mod); err != nil {
		log.Println("failed to write module to response:", err)
	}
}

func InfoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("===", r.URL, "===")
	unmapped, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mapped, major, err := MapPath(unmapped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if major != "" {
		mapped = mapped + "/" + major
	}

	version, err := module.UnescapeVersion(mux.Vars(r)["version"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	content, timestamp, err := source.GetInfo(mapped, version)
	if err != nil {
		if errors.Is(err, backend.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	data := struct {
		Version string
		Time    string
	}{
		Version: content,
		Time:    timestamp.UTC().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// nothing we can do now except for logging it
		log.Println("failed to write info content:", err)
	}
}

func ArchiveHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("===", r.URL, "===")
	unmapped := mux.Vars(r)["module"]
	unmapped, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mapped, major, err := MapPath(unmapped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if major != "" {
		mapped = mapped + "/" + major
	}

	version, err := module.UnescapeVersion(mux.Vars(r)["version"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmp, err := source.GetArchive(mapped, version)
	if err != nil {
		if errors.Is(err, backend.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("content-type", "application/zip")
	if _, err := io.Copy(w, tmp); err != nil {
		log.Println("failed to write zip archive to response:", err)
	}

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

//func NewRemote(url string) *git.Remote {
//	return git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
//		URLs: []string{url},
//	})
//}
//
//func StartAction(ctx *cli.Context) error {
//	//url := "https://github.com/llvm/llvm-project/llvm"
//	//tag := "llvmorg-14.0.3"
//
//	//_, filename := path.Split(url)
//	//filename = fmt.Sprintf("%s@%s.zip", filename, tag)
//	//f, err := os.Create(filename)
//	//if err != nil {
//	//	return errors.Wrapf(err, "failed to create file %s", filename)
//	//}
//	//defer f.Close()
//	//
//	//if err := BuildArchive(url, tag, f); err != nil {
//	//	os.Remove(filename)
//	//	return errors.Wrap(err, "failed to build archive")
//	//}
//
//	//tag = "llvmorg"
//	//tags, err := ListTags(url, tag)
//	//if err != nil {
//	//	return errors.Wrap(err, "failed to list versions")
//	//}
//	//
//	//for i, tag := range tags {
//	//	log.Println(i, tag)
//	//}
//
//	//
//	//log.Println("Cloning base repository")
//	//repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
//	//	URL:      "git@github.com:wandel/pdbgen.git",
//	//	Auth:     key,
//	//	Progress: os.Stdout,
//	//})
//	//if err != nil {
//	//	return errors.Wrap(err, "failed to clone repository")
//	//}
//
//	//log.Println("creating remote")
//	//remote, err := repo.CreateRemote(&config.RemoteConfig{
//	//	Name: "backup",
//	//	URLs: []string{"git@gitlab.com:bwandel/pdbgen.git"},
//	//})
//	//if err != nil {
//	//	return errors.Wrap(err, "failed to create remote")
//	//}
//	//
//	//log.Println("pushing to backup")
//	//if err := remote.Push(&git.PushOptions{
//	//	RemoteName: "backup",
//	//	Auth:       key,
//	//	Progress:   os.Stdout,
//	//}); err != nil {
//	//	return errors.Wrap(err, "failed to push to remote")
//	//}
//
//	return nil
//}
