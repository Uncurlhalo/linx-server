package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/zenazn/goji/web"
)

func fileServeHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	fileName := c.URLParams["name"]
	filePath := path.Join(Config.filesDir, fileName)

	err := checkFile(fileName)
	if err == NotFoundErr {
		notFoundHandler(c, w, r)
		return
	} else if err == BadMetadata {
		oopsHandler(c, w, r, RespAUTO, "Corrupt metadata.")
		return
	}

	if !Config.allowHotlink {
		referer := r.Header.Get("Referer")
		if referer != "" && !strings.HasPrefix(referer, Config.siteURL) {
			w.WriteHeader(403)
			return
		}
	}

	w.Header().Set("Content-Security-Policy", Config.fileContentSecurityPolicy)

	http.ServeFile(w, r, filePath)
}

var staticCache = make(map[string][]byte)

func staticHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path[len(path)-1:] == "/" {
		notFoundHandler(c, w, r)
		return
	} else {
		if path == "/favicon.ico" {
			path = "/static/images/favicon.gif"
		}

		filePath := strings.TrimPrefix(path, "/static/")

		_, exists := staticCache[filePath]
		if !exists {
			file, err := staticBox.Open(filePath)
			if err != nil {
				notFoundHandler(c, w, r)
				return
			}

			buf := bytes.NewBuffer(nil)
			io.Copy(buf, file)
			staticCache[filePath] = buf.Bytes()
		}

		w.Header().Set("Etag", timeStartedStr)
		w.Header().Set("Cache-Control", "max-age=86400")
		http.ServeContent(w, r, filePath, timeStarted, bytes.NewReader(staticCache[filePath]))
		return
	}
}

func checkFile(filename string) error {
	filePath := path.Join(Config.filesDir, filename)

	_, err := os.Stat(filePath)
	if err != nil {
		return NotFoundErr
	}

	expired, err := isFileExpired(filename)
	if err != nil {
		return err
	}

	if expired {
		os.Remove(path.Join(Config.filesDir, filename))
		os.Remove(path.Join(Config.metaDir, filename))
		return NotFoundErr
	}

	return nil
}
