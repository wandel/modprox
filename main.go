package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/urfave/cli"
	"io"
	"log"
	"net/http"
	"os"
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

func ListHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/list", vars["module"])
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(w, "failed to get list: %s", err)
		log.Println("failed to get list:", err)
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
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(w, "failed to get archive: %s", err)
		log.Println("failed to get archive:", err)
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

func ListenAction(ctx *cli.Context) error {
	router := mux.NewRouter()
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

//func StartAction(ctx *cli.Context) error {
//	key, err := ssh.NewPublicKeysFromFile("git", "c:\\users\\brett\\.ssh\\id_ed25519", "")
//	if err != nil {
//		return errors.Wrap(err, "failed to open private key")
//	}
//
//	log.Println("Cloning base repository")
//	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
//		URL:      "git@github.com:wandel/pdbgen.git",
//		Auth:     key,
//		Progress: os.Stdout,
//	})
//	if err != nil {
//		return errors.Wrap(err, "failed to clone repository")
//	}
//
//	log.Println("creating remote")
//	remote, err := repo.CreateRemote(&config.RemoteConfig{
//		Name: "backup",
//		URLs: []string{"git@gitlab.com:bwandel/pdbgen.git"},
//	})
//	if err != nil {
//		return errors.Wrap(err, "failed to create remote")
//	}
//
//	log.Println("pushing to backup")
//	if err := remote.Push(&git.PushOptions{
//		RemoteName: "backup",
//		Auth:       key,
//		Progress:   os.Stdout,
//	}); err != nil {
//		return errors.Wrap(err, "failed to push to remote")
//	}
//
//	return nil
//}
