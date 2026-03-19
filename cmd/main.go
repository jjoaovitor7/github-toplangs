package main

import (
  app "jjoaovitor7/github-toplangs/internal"
  "log"
  "net/http"
  "os"
)

import "github.com/newrelic/go-agent/v3/newrelic"

var PORT = os.Getenv("PORT")

func main() {
  if PORT == "" {
    PORT = ":8080"
  }

  app.SetTemplatesDir()

  mux := http.NewServeMux()
//  mux.HandleFunc("/", app.IndexRouteHandler)
//  mux.HandleFunc("/toplangs", app.TopLangsRouteHandler)

  nr, _ := newrelic.NewApplication(
    newrelic.ConfigAppName(os.Getenv("NEWRELIC_APPNAME")),
    newrelic.ConfigLicense(os.Getenv("NEWRELIC_LICENSEKEY")),
    newrelic.ConfigAppLogForwardingEnabled(true),
  )

  mux.HandleFunc(newrelic.WrapHandleFunc(nr, "/", app.IndexRouteHandler))
  mux.HandleFunc(newrelic.WrapHandleFunc(nr, "/toplangs", app.TopLangsRouteHandler))

  logging := app.LoggingMiddleware(mux)
  log.Printf("Server started at %s", PORT)
  if err := http.ListenAndServe(PORT, logging); err != nil {
    log.Fatal(err)
  }
}
