package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/exp/maps"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/wandel/modprox/backend"
	"github.com/wandel/modprox/utils"

	"golang.org/x/mod/module"
)

func main() {
	app := cli.NewApp()
	app.Name = "ModProx"
	app.Usage = "Golang Module Proxy"
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "address", Value: "127.0.0.1:8000", Usage: "the address to listen on"},
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	app.Action = func(ctx *cli.Context) error {
		//be := &backend.Git{
		//	CacheDir: "c:\\temp\\cache",
		//}
		//if err := be.Load(); err != nil {
		//	return errors.Wrap(err, "failed to load ")
		//}

		source := backend.NewMultiBackend(
			&backend.Direct{
				Environ: []string{
					"GOPROXY=direct",
					"GOPATH=c:/temp",
				},
			},
			//&backend.ModuleProxy{},
		)

		handler, err := NewHandler(source)
		if err != nil {
			return errors.Wrap(err, "failed to create modprox handler")
		}

		address := ctx.GlobalString("address")
		log.Println("listening on", address)
		if err := http.ListenAndServe(address, handler); err != nil {
			return err
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln("failed to run app:", err)
	}
}

func NewHandler(source backend.Backend) (http.Handler, error) {
	if source == nil {
		return nil, errors.New("source must not be nil")
	}

	h := Handler{
		source: source,
	}

	h.router = mux.NewRouter()
	h.router.HandleFunc("/", h.HandleStatus)
	h.router.HandleFunc("/{module:.+}/@v/list", h.HandleList)
	h.router.HandleFunc("/{module:.+}/@latest", h.HandleLatest)
	h.router.HandleFunc("/{module:.+}/@v/{version}.mod", h.HandleMod)
	h.router.HandleFunc("/{module:.+}/@v/{version}.info", h.HandleInfo)
	h.router.HandleFunc("/{module:.+}/@v/{version}.zip", h.HandleArchive)

	return &h, nil
}

type Handler struct {
	router *mux.Router
	source backend.Backend

	missing   sync.Map
	outofdate sync.Map
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	output := func(x sync.Map) []string {
		tmp := map[string]time.Time{}
		x.Range(func(key, value any) bool {
			tmp[key.(string)] = value.(time.Time)
			return true
		})

		keys := maps.Keys(tmp)
		sort.Strings(keys)
		return keys
	}

	io.WriteString(w, "<h1>Missing:</h1><ul>")
	for _, key := range output(h.missing) {
		io.WriteString(w, "<li>"+key+"</li>")

	}
	io.WriteString(w, "</ul><h1>Out Of Date:</h1><ul>")
	for _, key := range output(h.outofdate) {
		io.WriteString(w, "<li>"+key+"</li>")

	}
	io.WriteString(w, "</ul>")

}

// HandleList provides a list of valid versions (git tags) for a module
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	// validate the path first
	path := mux.Vars(r)["module"]
	if err := module.CheckPath(path); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// split the major version out
	prefix, major, _ := module.SplitPathVersion(path)

	// Check for modules
	if versions, err := h.source.GetList(prefix, major); err != nil {
		if errors.Is(err, backend.ErrNotFound) {
			// Do not sync on 404, as go get send a request for each part in the path.
			h.missing.Store(path, time.Now())
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

func (h *Handler) HandleLatest(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	// if 404, then add module to the "missing" list

	unmapped := mux.Vars(r)["module"]
	mapped, major, err := utils.MapPath(unmapped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	path, err := module.UnescapePath(mapped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	latest, timestamp, err := h.source.GetLatest(path, major)
	if err != nil {
		if errors.Is(err, backend.ErrOutOfDate) {
			h.outofdate.Store(path, time.Now())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else if errors.Is(err, backend.ErrNotFound) {
			h.missing.Store(path, time.Now())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		h.outofdate.Delete(path)
		h.missing.Delete(path)
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

func (h *Handler) HandleMod(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
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

	mod, err := h.source.GetModule(path, version)
	if err != nil {
		if errors.Is(err, backend.ErrOutOfDate) {
			h.outofdate.Store(path+"@"+version, time.Now())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else if errors.Is(err, backend.ErrNotFound) {
			h.missing.Store(path, time.Now())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		h.outofdate.Delete(path + "@" + version)
		h.missing.Delete(path)
	}

	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	if _, err := io.WriteString(w, mod); err != nil {
		// log.Println("failed to write module to response:", err)
	}
}

func (h *Handler) HandleInfo(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
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

	content, timestamp, err := h.source.GetInfo(path, version)
	if err != nil {
		if errors.Is(err, backend.ErrOutOfDate) {
			h.outofdate.Store(path+"@"+version, time.Now())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else if errors.Is(err, backend.ErrNotFound) {
			h.missing.Store(path, time.Now())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		h.outofdate.Delete(path + "@" + version)
		h.missing.Delete(path)
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
		// log.Println("failed to write info content:", err)
	}
}

func (h *Handler) HandleArchive(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
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

	tmp, err := h.source.GetArchive(path, version)
	if err != nil {
		if errors.Is(err, backend.ErrOutOfDate) {
			h.outofdate.Store(path+"@"+version, time.Now())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else if errors.Is(err, backend.ErrNotFound) {
			h.missing.Store(path, time.Now())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		h.outofdate.Delete(path + "@" + version)
		h.missing.Delete(path)
	}

	w.Header().Set("content-type", "application/zip")
	if _, err := io.Copy(w, tmp); err != nil {
		// log.Println("failed to write zip archive to response:", err)
	}
}
