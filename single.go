package main

//
//func main() {
//	app := cli.NewApp()
//	app.Action = ServeAction
//	if err := app.Run(os.Args); err != nil {
//		log.Fatalln("failed running the app:", err)
//	}
//}
//
//func ServeAction(ctx *cli.Context) error {
//	router := mux.NewRouter()
//	router.HandleFunc("/{module:.+}/@v/list", ListHandler)
//	router.HandleFunc("/{module:.+}/@latest", LatestHandler)
//	router.HandleFunc("/{module:.+}/@v/{version}.mod", ModHandler)
//	router.HandleFunc("/{module:.+}/@v/{version}.info", InfoHandler)
//	router.HandleFunc("/{module:.+}/@v/{version}.zip", ArchiveHandler)
//	router.HandleFunc("/", IndexHandler)
//	http.Handle("/", router)
//
//	if err := http.ListenAndServe("127.0.0.1:8000", nil); err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func IndexHandler(w http.ResponseWriter, r *http.Request) {
//	log.Println(r.URL)
//	w.WriteHeader(http.StatusOK)
//}
//
//func ListHandler(w http.ResponseWriter, r *http.Request) {
//	log.Println(r.URL)
//}
//
//func LatestHandler(w http.ResponseWriter, r *http.Request) {
//	log.Println(r.URL)
//}
//
//func ModHandler(w http.ResponseWriter, r *http.Request) {
//	log.Println(r.URL)
//}
//
//func InfoHandler(w http.ResponseWriter, r *http.Request) {
//	log.Println(r.URL)
//}
//
//func ArchiveHandler(w http.ResponseWriter, r *http.Request) {
//	log.Println(r.URL)
//}
