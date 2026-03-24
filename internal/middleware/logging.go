package middleware

import (
  "log"
  "net/http"
  "strings"
  "time"
)

type loggingWriter struct {
  http.ResponseWriter
  status int
}

func apacheLog(r *http.Request, status int) {
  ip := r.RemoteAddr
  if strings.Contains(ip, ":") {
    ip = strings.Split(ip, ":")[0]
  }

  // https://go.dev/src/time/format.go
  now := time.Now().Format("02/Jan/2006:15:04:05 -0300")

  log.Printf("%s - - [%s] \"%s %s %s\" %d\n",
    ip,
    now,
    r.Method,
    r.RequestURI,
    r.Proto,
    status,
  )
}

func LoggingMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    lw := &loggingWriter{w, http.StatusOK}
    next.ServeHTTP(lw, r)
    apacheLog(r, lw.status)
  })
}

func (lw *loggingWriter) WriteHeader(code int) {
  if lw.status != 0 {
    return
  }

  lw.status = code
  lw.ResponseWriter.WriteHeader(code)
}
