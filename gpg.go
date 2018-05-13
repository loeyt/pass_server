package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

type systemGpg struct {
	command string
}

func (s *systemGpg) encrypt(input string, ids ...string) (string, error) {
	args := []string{"--encrypt", "--armor"}
	for _, id := range ids {
		args = append(args, "--recipient", id)
	}
	return s.run(input, args...)
}

func (s *systemGpg) enarmor(input string) (string, error) {
	output, err := s.run(input, "--enarmor")
	if err != nil {
		return output, err
	}
	return strings.Replace(output, "PGP ARMORED FILE", "PGP MESSAGE", -1), nil
}

func (s *systemGpg) run(input string, args ...string) (string, error) {
	cmd := exec.Command(s.command, args...)
	w, err := cmd.StdinPipe()
	if err != nil {
		return "", errors.Wrap(err, "failed to open pipe")
	}
	_, err = fmt.Fprint(w, input)
	if err != nil {
		return "", errors.Wrap(err, "write failed")
	}
	err = w.Close()
	if err != nil {
		return "", errors.Wrap(err, "failed to close stdin")
	}
	b, err := cmd.Output()
	return string(b), errors.Wrap(err, "command failed")
}
