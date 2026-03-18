package main

import (
  app "jjoaovitor7/github-toplangs/internal"
  "log"
  "net/http"
  "os"
)

var PORT = os.Getenv("PORT")

func main() {
  if PORT == "" {
    PORT = ":8080"
  }

  app.SetTemplatesDir()

  mux := http.NewServeMux()
  mux.HandleFunc("/", app.IndexRouteHandler)
  mux.HandleFunc("/toplangs", app.TopLangsRouteHandler)
  logging := app.LoggingMiddleware(mux)
  log.Printf("Server started at %s", PORT)
  if err := http.ListenAndServe(PORT, logging); err != nil {
    log.Fatal(err)
  }
}
