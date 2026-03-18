package app

import (
  "bytes"
  "encoding/json"
  "errors"
  "fmt"
  "html/template"
  "log"
  "net/http"
  "os"
  "path/filepath"
  "sort"
  "strconv"
  "strings"
  "time"
)

type Repo struct {
  Name      string `json:"name"`
  CreatedAt string `json:"created_at"`
}

type SVGData struct {
  BgColor    string
  TitleColor string
  Width      int
  Height     int
  Langs      []LangView
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

var cache = map[string][]byte{}

var GITHUB_API = "https://api.github.com"

var (
  templatesDir     string
  topLangsTemplate *template.Template
  indexTemplate    *template.Template
)

func SetTemplatesDir() {
  templatesDir = os.Getenv("TEMPLATES_DIR")
  if templatesDir == "" {
    templatesDir = "templates"
  }

  topLangsTemplate = template.Must(template.ParseFiles(filepath.Join(templatesDir, "toplangs.tmpl")))
  indexTemplate = template.Must(template.ParseFiles(filepath.Join(templatesDir, "index.tmpl")))
}

func fetchRepos(username string, token string) ([]Repo, error, int) {
  url := fmt.Sprintf("%s/users/%s/repos?per_page=99", GITHUB_API, username)
  req, _ := http.NewRequest("GET", url, nil)
  req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

  res, err := http.DefaultClient.Do(req)
  if res.StatusCode == http.StatusNotFound {
    return nil, errors.New("User not found."), http.StatusNotFound
  }

  if err != nil {
    log.Println(err)
    return nil, errors.New("Error getting repositories."), http.StatusInternalServerError
  }
  defer res.Body.Close()

  var repos []Repo
  json.NewDecoder(res.Body).Decode(&repos)
  return repos, nil, http.StatusOK
}

func fetchLangs(username string, repo string, token string) (map[string]int, error, int) {
  url := fmt.Sprintf("%s/repos/%s/%s/languages", GITHUB_API, username, repo)
  req, _ := http.NewRequest("GET", url, nil)
  req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

  res, err := http.DefaultClient.Do(req)
  if err != nil {
    log.Println(err)
    return nil, errors.New("Error getting languages."), http.StatusInternalServerError
  }
  defer res.Body.Close()

  var langs map[string]int
  json.NewDecoder(res.Body).Decode(&langs)
  return langs, nil, http.StatusOK
}

func generateSVG(externalData struct {
  Query      string
  BgColor    string
  TitleColor string
}, langs []LangView) []byte {
  const (
    colWidth  = 128
    colLeftX  = 32
    colRightX = 128
    colStartY = 99
    colStepY  = 42
  )

  sum := 0
  for _, l := range langs {
    sum += l.Bytes
  }

  colLeft := (len(langs) + 1) / 2
  colRight := len(langs) - colLeft
  maxRows := colLeft

  if colRight > maxRows {
    maxRows = colRight
  }

  colX := []int{colLeftX, colRightX + 64}
  percentX := []int{colLeftX + colWidth, colRightX + colWidth + 64}

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
    BgColor:    externalData.BgColor,
    TitleColor: externalData.TitleColor,
    Width:      364,
    Height:     colStartY + colStepY*maxRows + 16,
    Langs:      langs,
  }

  var svgBuffer bytes.Buffer
  topLangsTemplate.Execute(&svgBuffer, data)
  svg := svgBuffer.Bytes()
  cache[externalData.Query] = svg
  return svg
}

func topLangsHandler(w http.ResponseWriter, username string, token string, limit int, hide map[string]bool) map[string]int {
  sum := make(map[string]int)
  repos, err, status := fetchRepos(username, token)

  if err != nil {
    log.Println(err)
    http.Error(w, err.Error(), status)
    return nil
  }

  for _, r := range repos {
    langs, err, status := fetchLangs(username, r.Name, token)
    if err != nil {
      log.Println(err)
      http.Error(w, err.Error(), status)
      return nil
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
  return sum
}

func TopLangsRouteHandler(w http.ResponseWriter, r *http.Request) {
  token := os.Getenv("GITHUB_TOKEN")

  user := r.URL.Query().Get("user")
  if user == "" {
    http.Error(w, "Field 'user' is empty.", http.StatusBadRequest)
    return
  }

  bgColor := r.URL.Query().Get("bgcolor")
  if bgColor == "" {
    bgColor = "111"
  }

  titleColor := r.URL.Query().Get("titlecolor")
  if titleColor == "" {
    titleColor = "fff"
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

  url := fmt.Sprintf("%s/users/%s", GITHUB_API, user)
  req, _ := http.NewRequest("GET", url, nil)
  req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

  res, _ := http.DefaultClient.Do(req)
  if res.StatusCode == 404 {
    http.Error(w, "User not found.", http.StatusNotFound)
    return
  }

  query := r.URL.RawQuery
  if svg, ok := cache[query]; ok {
    w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write(svg)
    return
  }

  data := topLangsHandler(w, user,
    token,
    limit,
    hide)

  var list []LangView
  for k, v := range data {
    list = append(list, LangView{
      Name:  k,
      Bytes: v,
    })
  }

  sort.Slice(list, func(i, j int) bool {
    return list[i].Bytes > list[j].Bytes
  })

  svg := generateSVG(struct {
    Query      string
    BgColor    string
    TitleColor string
  }{
    Query:      query,
    BgColor:    fmt.Sprintf("#%s", bgColor),
    TitleColor: fmt.Sprintf("#%s", titleColor),
  }, list)

  w.Header().Set("Cache-Control", "public, max-age=43200, must-revalidate")
  w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
  w.Header().Set("Content-Length", strconv.Itoa(len(svg)))

  w.WriteHeader(http.StatusOK)
  w.Write(svg)
}

func IndexRouteHandler(w http.ResponseWriter, r *http.Request) {
  indexTemplate.Execute(w, nil)
  w.WriteHeader(http.StatusOK)
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
    lrw := &responseWriter{w, http.StatusOK}
    next.ServeHTTP(lrw, r)
    apacheLog(r, lrw.status)
  })
}

func (rw *responseWriter) WriteHeader(code int) {
  rw.status = code
  rw.ResponseWriter.WriteHeader(code)
}
