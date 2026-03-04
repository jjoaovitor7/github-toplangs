package main

import (
  "net/http"
  "fmt"
  "log"
  "encoding/json"
  "sort"
  "html/template"
  "time"
  "strings"
  "strconv"
  "os"
)

type Repo struct {
  Name string `json:"name"`
  CreatedAt string `json:"created_at"`
}

type SVGData struct {
  Width int
  Height int
  Langs []LangView
}

type LangView struct {
  Name     string
  NameY    int
  Percent  float64
  PercentX float64
  Width    float64
  BarY     int
  Color    string
  Bytes    int
  ColX     int
}

type responseWriter struct {
  http.ResponseWriter
  status int
}

var svgTemplate = template.Must(
  template.ParseFiles("templates/toplangs.tmpl"),
)

// https://raw.githubusercontent.com/github/linguist/master/lib/linguist/languages.yml
var langColors = map[string]string{
  "Go":         "#00ADD8",
  "JavaScript": "#f1e05a",
  "TypeScript": "#3178c6",
  "PHP":        "#4F5D95",
  "Lua":        "#000080",
  "Java":       "#b07219",
  "Kotlin":     "#A97BFF",
  "HTML":       "#e34c26",
  "CSS":        "#663399",
  "Svelte":     "#ff3e00",
  "MDX":        "#fcb32c",
  "Shell":      "#89e051",
  "C++":        "#f34b7d",
  "C#":         "#178600",
  "Python":     "#3572A5",
}

const PORT = ":8000"

func fetchRepos(username, token string) ([]Repo, error) {
  url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100", username)

  req, _ := http.NewRequest("GET", url, nil)
  req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

  res, err := http.DefaultClient.Do(req)
  if err != nil {
    return nil, err
  }
  defer res.Body.Close()

  var repos []Repo
  json.NewDecoder(res.Body).Decode(&repos)
  return repos, nil
}

func fetchLanguages(username, repo, token string) (map[string]int, error) {
  url := fmt.Sprintf("https://api.github.com/repos/%s/%s/languages", username, repo)
  req, _ := http.NewRequest("GET", url, nil)
  req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

  res, err := http.DefaultClient.Do(req)
  if err != nil {
    return nil, err
  }
  defer res.Body.Close()

  var langs map[string]int
  json.NewDecoder(res.Body).Decode(&langs)
  return langs, nil
}

func topLangsHandler(username string, token string, limit int, hide map[string]bool,) (map[string]int, error) {
  sum := make(map[string]int)
  repos, err := fetchRepos(username, token)

  if (err != nil) {
    return sum, err
  }

  for _, r := range repos {
    langs, err := fetchLanguages(username, r.Name, token)
    if err != nil {
      return sum, err
    }

    for lang, bytes := range langs {
      lang = strings.ReplaceAll(lang, "#", "sharp")
      lang = strings.ReplaceAll(lang, "++", "pp")
      hideLang := strings.ToLower(lang)

      if !hide[hideLang] {
        sum[lang] += bytes
      }
    }
  }

  if limit > 0 && limit < len(sum) {
    keys := make([]string, 0, len(sum))
    for k := range sum {
      keys = append(keys, k)
    }

    sort.Slice(keys, func(i int, j int) bool {
      return sum[keys[i]] > sum[keys[j]]
    })

    topLangs := make(map[string]int, limit)
    for _, k := range keys[:limit] {
      topLangs[k] = sum[k]
    }
    sum = topLangs
  }
  return sum, nil
}

func renderSVG(w http.ResponseWriter, langs []LangView) {
  const (
    colWidth  = 128
    colLeftX  = 16
    colRightX = 128
    colStartY = 99
    colStepY  = 42
  )

  w.Header().Set("Content-Type", "image/svg+xml")
  w.WriteHeader(http.StatusOK)

  sum := 0
  for _, l := range langs {
    sum += l.Bytes
  }

  colLeft  := (len(langs) + 1) / 2
  colRight := len(langs) - colLeft
  maxRows  := colLeft

  if colRight > maxRows {
    maxRows = colRight
  }

  colX := []int{
    colLeftX,
    colRightX + 64
  }

  percentX := []int{
    colLeftX + colWidth,
    colRightX + colWidth + 64
  }

  for i := range langs {
    l := &langs[i]
    l.Percent = float64(l.Bytes) / float64(sum) * 100
    l.Width = (l.Percent / 100) * colWidth

    color := langColors[l.Name]
    if l.Color == "" {
      l.Color = "#ffffff"
    }
    l.Color = color

    col := 0
    row := i
    if i >= colLeft {
      col = 1
      row = i - colLeft
    }

    l.NameY = colStartY + row*colStepY
    l.BarY = l.NameY + 4
    l.ColX = colX[col]
    l.PercentX = float64(percentX[col])
  }

  data := SVGData{
    Width: 432,
    Height: colStartY + colStepY*maxRows + 16,
    Langs: langs,
  }
  svgTemplate.Execute(w, data)
}

func handler(w http.ResponseWriter, r *http.Request) {
  token := os.Getenv("GITHUB_TOKEN")

  user := r.URL.Query().Get("user")
  if user == "" {
    http.Error(w, "Field 'user' is empty.", http.StatusBadRequest)
  }

  limitQuery := r.URL.Query().Get("limit")
  var limit int = 8
  if n, err := strconv.Atoi(limitQuery); err == nil {
    limit = n
  }

  hideQuery := r.URL.Query().Get("hide")
  hide := map[string]bool{}
  if hideQuery != "" {
    for _, h := range strings.Split(hideQuery, ",") {
      hideLang := strings.TrimSpace(strings.ToLower(h))
      hide[hideLang] = true
    }
  }

  data, _ := topLangsHandler(user,
    token,
    limit,
    hide,
  )

  var list []LangView
  for k, v := range data {
    list = append(list, LangView{
      Name: k,
      Bytes: v,
    })
  }

  sort.Slice(list, func(i,j int) bool {
    return list[i].Bytes > list[j].Bytes
  })

  w.Header().Set("Content-Type", "image/svg+xml")
  renderSVG(w, list)
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

func loggingMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    lrw := &responseWriter{w, http.StatusOK}
    next.ServeHTTP(lrw, r)
    apacheLog(r, lrw.status)
  })
}

func (rw *responseWriter) WriteHeader(code int) {
  rw.status = code
  rw.ResponseWriter.WriteHeader(code)
}

func main() {
  mux := http.NewServeMux()
  mux.HandleFunc("/toplangs", handler)
  logging := loggingMiddleware(mux)
  log.Printf("Server started at %s", PORT)
  http.ListenAndServe(PORT, logging)
}

