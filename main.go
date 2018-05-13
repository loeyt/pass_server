package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp/armor"
)

type secretName struct {
	Path     string `json:"path"`
	Username string `json:"username"`
}

func wrapResponse(input string) (string, error) {
	b, err := json.Marshal(struct {
		Response string `json:"response"`
	}{
		Response: input,
	})
	return string(b), err
}

type gpg interface {
	encrypt(string, ...string) (string, error)
	enarmor(string) (string, error)
}

type indexer func(dirname string) (ids []string, index string, secrets map[secretName]string, err error)

type serverBuilder struct {
	dirname string
	gpg
	indexer
}

func (b *serverBuilder) build() (*server, error) {
	ids, index, secrets, err := b.indexer(b.dirname)
	if err != nil {
		return nil, errors.Wrap(err, "indexer failed")
	}
	index, err = b.encrypt(index, ids...)
	if err != nil {
		return nil, errors.Wrap(err, "index encrypt failed")
	}
	index, err = wrapResponse(index)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap index")
	}

	for i := range secrets {
		secrets[i], err = b.enarmor(secrets[i])
		if err != nil {
			return nil, errors.Wrap(err, "enarmor failed")
		}
		secrets[i], err = wrapResponse(secrets[i])
		if err != nil {
			return nil, errors.Wrap(err, "failed to wrap secret")
		}
	}
	return &server{
		index:   index,
		secrets: secrets,
	}, nil
}

func enarmor(r io.Reader) ([]byte, error) {
	buf := &bytes.Buffer{}
	w, err := armor.Encode(buf, "PGP MESSAGE", nil)
	if err != nil {
		return buf.Bytes(), errors.Wrap(err, "failed to create armor encoder")
	}
	_, err = io.Copy(w, r)
	if err != nil {
		return buf.Bytes(), errors.Wrap(err, "failed to copy to armored encoder")
	}
	err = w.Close()
	return buf.Bytes(), errors.Wrap(err, "failed to close armored encoder")
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:7277", "address to listen on")
	store := fs.String("store", os.ExpandEnv("$HOME/.password-store"), "location of the password store")
	gpg := fs.String("gpg", "gpg", "gpg command")
	_ = fs.Parse(os.Args[1:])

	cmd, err := exec.LookPath(*gpg)
	if err != nil {
		return errors.Wrap(err, "failed to find command")
	}

	builder := &serverBuilder{
		dirname: *store,
		gpg: &systemGpg{
			command: cmd,
		},
		indexer: passIndexer,
	}
	server, err := builder.build()
	if err != nil {
		return errors.Wrap(err, "failed to build server")
	}
	c := make(chan os.Signal, 5)
	signal.Notify(c, syscall.SIGUSR1)
	rh := &replacingHandler{
		Handler: server.handler(),
	}
	go func() {
		for {
			select {
			case <-c:
				log.Printf("caught SIGUSR1, reloading")
				server, err := builder.build()
				if err != nil {
					log.Printf("failed to reload: %v", err)
				} else {
					rh.replace(server.handler())
					log.Printf("reload successful")
				}
			}
		}
	}()
	log.Printf("starting server at %q", *addr)
	err = http.ListenAndServe(*addr, rh)
	return err
}
