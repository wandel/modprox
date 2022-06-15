package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/wandel/modprox/backend"
	"github.com/wandel/modprox/utils"
	"golang.org/x/mod/module"
)

var source backend.Backend

func main() {
	app := cli.NewApp()
	app.Name = "ModProx"
	app.Usage = "Golang Module Proxy"
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "address", Value: "127.0.0.1:8000", Usage: "the address to listen on"},
		&cli.StringFlag{Name: "username", Value: "git", Usage: "the username to use"},
		&cli.StringFlag{Name: "password", Value: "", Usage: "the password to use"},
		&cli.StringFlag{Name: "privatekey", Value: "", Usage: "the ssh private key to use"},
		cli.StringFlag{Name: "token", Value: "", Usage: "gitlab personal access token"},
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	app.Before = func(ctx *cli.Context) error {
		source = backend.NewMultiBackend(&backend.Direct{})
		return nil
	}

	app.Action = func(ctx *cli.Context) error {
		router := mux.NewRouter()
		router.HandleFunc("/{module:.+}/@v/list", ListHandler)
		router.HandleFunc("/{module:.+}/@latest", LatestHandler)
		router.HandleFunc("/{module:.+}/@v/{version}.mod", ModHandler)
		router.HandleFunc("/{module:.+}/@v/{version}.info", InfoHandler)
		router.HandleFunc("/{module:.+}/@v/{version}.zip", ArchiveHandler)

		address := ctx.GlobalString("address")
		log.Println("listening on", address)
		if err := http.ListenAndServe(address, router); err != nil {
			return err
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln("failed to run app:", err)
	}
}

// ListHandler provides a list of valid versions (git tags) for a module
func ListHandler(w http.ResponseWriter, r *http.Request) {
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
	if versions, err := source.GetList(prefix, major); err != nil {
		if errors.Is(err, backend.ErrNotFound) {
			// Do not sync on 404, as go get send a request for each part in the path.
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
	log.Println(r.URL)
	// if 404, then add module to the "missing" list

	unmapped := mux.Vars(r)["module"]
	mapped, major, err := utils.MapPath(unmapped)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	// log.Println("===", unmapped, "===")

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
	log.Println(r.URL)
	unmapped, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// log.Println("===", unmapped, "===")

	mapped, major, err := utils.MapPath(unmapped)
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
		// log.Println("failed to write module to response:", err)
	}
}

func InfoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	unmapped, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// log.Println("===", unmapped, "===")

	mapped, major, err := utils.MapPath(unmapped)
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
		// log.Println("failed to write info content:", err)
	}
}

func ArchiveHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	unmapped := mux.Vars(r)["module"]
	unmapped, err := module.UnescapePath(mux.Vars(r)["module"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// log.Println("===", unmapped, "===")

	mapped, major, err := utils.MapPath(unmapped)
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

	tmp, err := source.GetArchive(unmapped, version)
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
		// log.Println("failed to write zip archive to response:", err)
	}
}
