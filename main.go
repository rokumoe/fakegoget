package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"regexp"
	"sync"
	"syscall"
	"text/template"
)

type Meta struct {
	re         *regexp.Regexp
	Pattern    string `json:"pattern"`
	Pkg        string `json:"pkg"`
	VCS        string `json:"vcs"`
	Repo       string `json:"repo"`
	Source     string `json:"source"`
	SourceDir  string `json:"sourcedir"`
	SourceLine string `json:"sourceline"`
	Doc        string `json:"doc"`
	Body       string `json:"body"`
}

var metatpl = template.Must(template.New("meta").Parse(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.Pkg}} {{.VCS}} {{.Repo}}"/>
{{- if .Source}}
<meta name="go-source" content="{{.Pkg}} {{.Source}} {{.SourceDir}} {{.SourceLine}}"/>
{{- end}}
{{- if .Doc}}
<meta http-equiv="refresh" content="0; url={{.Doc}}"/>
{{- end}}
</head>
<body>
{{- if .Body}}
{{.Body}}
{{- end}}
</body>
</html>
`))

var pkgs struct {
	sync.Mutex
	m map[string]*Meta
}

func getpkg(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.FormValue("go-get") != "1" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	pkg := path.Join(r.Host, r.URL.Path)

	log.Printf("get %s", pkg)

	pkgs.Lock()
	meta := pkgs.m[pkg]
	if meta == nil {
		for _, m := range pkgs.m {
			if m.re != nil && m.re.MatchString(pkg) {
				meta = m
				break
			}
		}
	}
	pkgs.Unlock()

	if meta == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	metatpl.Execute(w, meta)
}

func loadmeta(filename string) (map[string]*Meta, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var allmeta []*Meta
	err = json.Unmarshal(data, &allmeta)
	if err != nil {
		return nil, err
	}
	meta := make(map[string]*Meta, len(allmeta))
	for _, m := range allmeta {
		if m.Pattern != "" {
			m.re = regexp.MustCompile(m.Pattern)
		}
		meta[m.Pkg] = m
	}
	return meta, nil
}

func main() {
	var (
		serve  string
		cert   string
		key    string
		config string
	)
	flag.StringVar(&serve, "serve", "127.0.0.1:443", "serve")
	flag.StringVar(&cert, "cert", "./server.crt", "cert")
	flag.StringVar(&key, "key", "./server.key", "key")
	flag.StringVar(&config, "config", "./config.json", "config")
	flag.Parse()

	meta, err := loadmeta(config)
	if err != nil {
		log.Fatalln(err)
	}
	pkgs.m = meta
	go func() {
		err = http.ListenAndServeTLS(serve, cert, key, http.HandlerFunc(getpkg))
		if err != nil {
			log.Fatalln(err)
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)
	for range signals {
		meta, err := loadmeta(config)
		if err != nil {
			log.Printf("loadmeta: %s", err)
		} else {
			pkgs.Lock()
			pkgs.m = meta
			pkgs.Unlock()
		}
	}
}
