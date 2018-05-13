package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func passIndexer(dirname string) ([]string, string, map[secretName]string, error) {
	ids, err := readIDs(dirname)
	if err != nil {
		return nil, "", nil, errors.Wrap(err, "failed to read .gpg-id")
	}
	files := make([]string, 0, 256)
	err = filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Dir(path) == dirname {
			return nil
		}
		if match, _ := filepath.Match("*.gpg", info.Name()); match {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return ids, "", nil, errors.Wrap(err, "dirwalk failed")
	}
	type item struct {
		Domain             string `json:"domain"`
		Path               string `json:"path"`
		Username           string `json:"username"`
		UsernameNormalized string `json:"username_normalized"`
	}
	list := make([]item, 0, len(files))
	secrets := make(map[secretName]string)
	for _, filename := range files {
		secret := strings.TrimSuffix(strings.TrimPrefix(filename, dirname), ".gpg")
		username := path.Base(secret)
		secret = strings.TrimPrefix(path.Dir(secret), "/")
		domain := path.Base(secret)
		list = append(list, item{
			Domain:             domain,
			Path:               secret,
			Username:           username,
			UsernameNormalized: normalize(username),
		})
		contents, err := ioutil.ReadFile(filename)
		if err != nil {
			return ids, "", nil, errors.Wrap(err, "readfile failed")
		}
		secrets[secretName{
			Path:     secret,
			Username: username,
		}] = string(contents)
	}
	b, err := json.Marshal(list)
	return ids, string(b), secrets, err
}

func readIDs(dirname string) ([]string, error) {
	ids := make([]string, 0, 8)
	f, err := os.Open(path.Join(dirname, ".gpg-id"))
	if err != nil {
		return nil, errors.Wrap(err, "open failed")
	}
	s := bufio.NewScanner(f)
	for s.Scan() {
		id := s.Text()
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	if s.Err() != nil {
		return ids, errors.Wrap(s.Err(), "scan failed")
	}
	err = f.Close()
	return ids, errors.Wrap(err, "close failed")
}

func normalize(s string) string {
	rs, n, err := transform.String(norm.NFKD, s)
	if err != nil {
		return ""
	}
	b := make([]byte, 0, n)
	for i := range rs {
		if rs[i] < 0x80 {
			b = append(b, rs[i])
		}
	}
	return string(b)
}
