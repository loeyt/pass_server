package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
)

// server serves a fully pre-calculated set of response bodies
type server struct {
	index   string                // json body for /secrets/ requests
	secrets map[secretName]string // json bodies for /secret/ requests
}

func (s *server) handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/secret/", jsonPostOnly(s.secretHandler()))
	m.HandleFunc("/secrets/", jsonPostOnly(s.indexHandler()))
	return m
}

func (s *server) indexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := &bytes.Buffer{}
		_, err := io.Copy(body, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, jsonError(err.Error()))
			return
		}
		err = r.Body.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, jsonError(err.Error()))
			return
		}
		io.WriteString(w, s.index)
	}
}

func (s *server) secretHandler() http.HandlerFunc {
	type request struct {
		Path     string      `json:"path"`
		Username interface{} `json:"username"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		body := &bytes.Buffer{}
		_, err := io.Copy(body, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, jsonError(err.Error()))
			return
		}
		err = r.Body.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, jsonError(err.Error()))
			return
		}
		d := json.NewDecoder(body)
		var v request
		err = d.Decode(&v)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, jsonError(err.Error()))
			return
		}
		var username string
		switch u := v.Username.(type) {
		case float64:
			username = strconv.FormatFloat(u, 'f', -1, 64)
		case string:
			username = u
		default:
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, jsonError("bad username"))
			return
		}
		secret, ok := s.secrets[secretName{
			Path:     v.Path,
			Username: username,
		}]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, jsonError("unknown secret"))
			return
		}
		io.WriteString(w, secret)
	}
}

// jsonPostOnly middleware denies non-JSON Content-Type and non-POST requests,
// and sets the response's Content-Type to JSON.
func jsonPostOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.Header().Add("Allow", "POST")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}
		if mediaType != "application/json" {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		h(w, r)
	}
}

func jsonError(error string) string {
	v := struct {
		Error string `json:"error"`
	}{
		Error: error,
	}
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
