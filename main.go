package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/xanzy/go-gitlab"
	module "golang.org/x/mod/module"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const GITLAB_TOKEN = ""

func main() {
	app := cli.NewApp()
	app.Name = "ModProx"
	app.Usage = "Golang Module Proxy"
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "username", Value: "git", Usage: "the username to use"},
		&cli.StringFlag{Name: "password", Value: "", Usage: "the password to use"},
		&cli.StringFlag{Name: "privatekey", Value: "", Usage: "the ssh private key to use"},
	}
	app.Commands = []cli.Command{
		cli.Command{Name: "serve", Action: ServeAction},
		cli.Command{Name: "test", Action: TestAction},
		//cli.Command{Name: "sync", Action: SyncAction, Flags: []cli.Flag{
		//	&cli.StringFlag{Name: "source, s", Usage: "the source repository to sync from"},
		//	&cli.StringFlag{Name: "destination, d", Usage: "the destination repository to sync to"},
		//}},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln("failed to run app:", err)
	}
}

func ServeAction(ctx *cli.Context) error {
	router := mux.NewRouter()
	router.HandleFunc("/{module:.+}/@v/list", ListHandler)
	router.HandleFunc("/{module:.+}/@latest", LatestHandler)
	router.HandleFunc("/{module:.+}/@v/{version}.mod", ModHandler)
	router.HandleFunc("/{module:.+}/@v/{version}.info", InfoHandler)
	router.HandleFunc("/{module:.+}/@v/{version}.zip", ArchiveHandler)
	http.Handle("/", router)

	if err := http.ListenAndServe("127.0.0.1:8000", nil); err != nil {
		return err
	}

	return nil
}

func TestAction(ctx *cli.Context) error {
	sha := "f79a8a8ca69d"
	client, err := gitlab.NewClient(GITLAB_TOKEN)
	if err != nil {
		return errors.Wrap(err, "failed to connect to github")
	}

	if _, resp, err := client.Repositories.Archive("mirror8/github.com/cpuguy83/go-md2man", &gitlab.ArchiveOptions{
		Format: gitlab.String("zip"),
		SHA:    gitlab.String(sha),
	}, nil); err != nil {
		return errors.Wrap(err, "failed to fetch archive")
	} else {
		log.Println(resp.StatusCode, '-', resp.Status)
	}

	return nil
}

//
//func getAuth(ctx *cli.Context) (transport.AuthMethod, error) {
//	username := ctx.GlobalString("username")
//	password := ctx.GlobalString("password")
//	privatekey := ctx.GlobalString("privatekey")
//
//	if "privatekey" != "" {
//		log.Println(privatekey, username, password)
//		auth, err := ssh.NewPublicKeysFromFile(username, privatekey, password)
//		if err != nil {
//			return nil, errors.Wrapf(err, "failed to load ssh key from '%s'", privatekey)
//		}
//		return auth, nil
//	} else if password != "" {
//		log.Println(username, password)
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

func MapPath(path string) string {
	// not sure if this is required...
	mappings := map[string]string{
		"google.golang.org/grpc":         "github.com/grpc/grpc-go",
		"google.golang.org/protobuf":     "github.com/protocolbuffers/protobuf-go",
		"google.golang.org/api":          "github.com/googleapis/google-api-go-client",
		"google.golang.org/genproto":     "github.com/googleapis/go-genproto",
		"cloud.google.com/go":            "github.com/googleapis/google-cloud-go",
		"go.opentelemetry.io/proto/otlp": "github.com/open-telemetry/opentelemetry-proto-go",
		"google.golang.org/appengine":    "github.com/golang/appengine",
		"honnef.co/go/tools":             "github.com/dominikh/go-tools",
	}

	if strings.HasPrefix(path, "gopkg.in/") {
		path = strings.TrimPrefix(path, "gopkg.in/")
		version := ""
		if parts := strings.Split(path, "."); len(parts) == 2 {
			path = parts[0]
			version = "/" + parts[1]
		}

		if parts := strings.Split(path, "/"); len(parts) == 1 {
			path = "go-" + parts[0] + "/" + parts[0]
		}

		return "github.com/" + path + version
	} else if strings.HasPrefix(path, "golang.org/x") {
		return strings.Replace(path, "golang.org/x", "github.com/golang", 1)
	}

	path, version, _ := module.SplitPathVersion(path)
	if value, ok := mappings[path]; ok {
		path = value
	}

	return path + version
}

// ListHandler provides a list of valid versions (git tags) for a module
func ListHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("===", r.URL, "===")
	path, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// use go's internal tooling to split the path into module and version
	if err := module.CheckPath(path); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name, version, _ := module.SplitPathVersion(path)
	log.Printf("list: path='%s', name='%s', version='%s'\n", path, name, version)

	client, err := gitlab.NewClient(GITLAB_TOKEN)
	if err != nil {
		http.Error(w, errors.Wrap(err, "failed to connect to gitlab").Error(), http.StatusInternalServerError)
	}

	// sync repository
	// TODO trigger and wait for a sync

	// fetch a list of tags
	tags, _, err := client.Tags.ListTags("mirror8/"+name, &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 1000, // TODO should do this properly in the future

		},
	}, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to get a list of tags: %s", err)
		return
	}

	// filter tag based on version
	for _, tag := range tags {
		// go helpfully provides a simple function to validate a path and version (tag) combination.
		if err := module.Check(path, tag.Name); err == nil {
			fmt.Fprintln(w, tag.Name)
		}
	}
}

func LatestHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("===", r.URL, "===")
	path, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// use go's internal tooling to split the path into module and version
	if err := module.CheckPath(path); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name, version, _ := module.SplitPathVersion(path)
	log.Printf("list: path='%s', name='%s', version='%s'\n", path, name, version)

	client, err := gitlab.NewClient(GITLAB_TOKEN)
	if err != nil {
		http.Error(w, errors.Wrap(err, "failed to connect to gitlab").Error(), http.StatusInternalServerError)
	}

	// sync repository
	// TODO trigger and wait for a sync

	// fetch a list of tags
	tags, _, err := client.Tags.ListTags("mirror8/"+name, &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 1000, // TODO should do this properly in the future

		},
	}, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to get a list of tags: %s", err)
		return
	}

	// filter tag based on version
	var latest *gitlab.Tag
	for _, tag := range tags {
		// go helpfully provides a simple function to validate a path and version (tag) combination.
		if err := module.Check(path, tag.Name); err != nil {
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
		http.Error(w, "failed to find a valid version", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Version string
		Time    string
	}{
		Version: latest.Name,
		Time:    latest.Commit.CommittedDate.UTC().Format(time.RFC3339),
	})

}

func ModHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("===", r.URL, "===")
	path, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	version, err := module.UnescapeVersion(mux.Vars(r)["version"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// use go's internal tooling to split the path into module and version
	if err := module.Check(path, version); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	commit := version
	if module.IsPseudoVersion(version) {
		if tmp, err := module.PseudoVersionRev(version); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			log.Println("commit:", tmp)
			commit = tmp
		}
	}

	name, _, _ := module.SplitPathVersion(path)
	log.Printf("mod: path='%s', name='%s', version='%s', commit='%s'\n", path, name, version, commit)

	client, err := gitlab.NewClient(GITLAB_TOKEN)
	if err != nil {
		http.Error(w, errors.Wrap(err, "failed to connect to gitlab").Error(), http.StatusInternalServerError)
	}

	if content, resp, err := client.RepositoryFiles.GetRawFile("mirror8/"+name, "go.mod", &gitlab.GetRawFileOptions{
		Ref: gitlab.String(commit),
	}); err != nil {
		_, major, _ := module.SplitPathVersion(path)
		if resp.StatusCode == http.StatusNotFound && major == "" {
			// generate a synthetic go.mod file if one does not exist (only appropriate for v0/v1)
			fmt.Fprintln(w, "module", path)
			return
		}
		http.Error(w, err.Error(), resp.StatusCode)
	} else if _, err := w.Write(content); err != nil {
		// nothing we can do now except for logging it
		log.Println("failed to write module content:", err)
	}
}

func InfoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("===", r.URL, "===")
	path, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	version, err := module.UnescapeVersion(mux.Vars(r)["version"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// use go's internal tooling to split the path into module and version
	if err := module.Check(path, version); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name, _, _ := module.SplitPathVersion(path)
	log.Printf("info: path='%s', name='%s', version='%s'\n", path, name, version)

	client, err := gitlab.NewClient(GITLAB_TOKEN)
	if err != nil {
		http.Error(w, errors.Wrap(err, "failed to connect to gitlab").Error(), http.StatusInternalServerError)
	}

	tag, resp, err := client.Tags.GetTag("mirror8/"+name, version, nil)
	if err != nil {
		http.Error(w, err.Error(), resp.StatusCode)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	data := struct {
		Version string
		Time    string
	}{
		Version: tag.Name,
		Time:    tag.Commit.CommittedDate.UTC().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// nothing we can do now except for logging it
		log.Println("failed to write info content:", err)
	}
}

func ArchiveHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("===", r.URL, "===")
	path, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	version, err := module.UnescapeVersion(mux.Vars(r)["version"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// use go's internal tooling to split the path into module and version
	if err := module.Check(path, version); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	commit := version
	if module.IsPseudoVersion(version) {
		if tmp, err := module.PseudoVersionRev(version); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			commit = tmp
		}
	}

	name, _, _ := module.SplitPathVersion(path)
	log.Printf("archive: path='%s', name='%s', version='%s', commit='%s'\n", path, name, version, commit)

	client, err := gitlab.NewClient(GITLAB_TOKEN)
	if err != nil {
		http.Error(w, errors.Wrap(err, "failed to connect to gitlab").Error(), http.StatusInternalServerError)
		return
	}

	var buffer bytes.Buffer
	if resp, err := client.Repositories.StreamArchive("mirror8/"+name, &buffer, &gitlab.ArchiveOptions{
		Format: gitlab.String("zip"),
		SHA:    gitlab.String(commit),
	}, nil); err != nil {
		http.Error(w, err.Error(), resp.StatusCode)
		return
	}

	reader := bytes.NewReader(buffer.Bytes())
	input, err := zip.NewReader(reader, reader.Size())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	path, err = module.EscapePath(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	version, err = module.EscapeVersion(version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	base := path + "@" + version + "/"
	log.Println(base)

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
				// we leave out any vendored file other than modules.txt
				continue
			}
		}

		// need to replace the base folder name with the correct module path (ie "github.com/urfave/cli/v2@v2.6.0")
		tmp := strings.Replace(file.Name, replace, base, 1)
		dst, err := output.Create(tmp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		src, err := file.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(dst, src); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			src.Close()
			return
		}
		src.Close()
	}

	output.Close()
	if _, err := io.Copy(w, &buffer1); err != nil {
		// not much else we can do
		log.Println("failed to copy final zip:", err)
	}
}