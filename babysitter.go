package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	killOn, command, err := parseArgs()
	if err != nil {
		if err == errGaveUsage {
			return
		}
		log.Fatal(err)
	}

	// outputC is checked below for a close() signal to stop the babysitter
	outputC := make(chan []byte)
	errorC := make(chan error)

	// start the babysat program
	go babysit(command, []byte(killOn), outputC, errorC)

	// forever wait for output, close, or error
	for {
		select {
		case err := <-errorC:
			fmt.Printf("Error: %s\n", err)
		case output, open := <-outputC:
			if !open {
				fmt.Println("Output closed")
				return
			}
			fmt.Printf("Output: %s\n", string(output))
		}
	}
}

func babysit(
	command []string, killOn []byte, outputC chan []byte, errorC chan error) {
	//nolint: gosec // this tool is run with locally a specified sub-command
	cmd := exec.Command(command[0], command[1:]...)
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		errorC <- err
		return
	}
	if err = cmd.Start(); err != nil {
		errorC <- err
		return
	}
	go listenAndKill(cmd, killOn, stdOut, outputC, errorC)
}

func listenAndKill(
	cmd *exec.Cmd, killOn []byte, stdOut io.Reader,
	outputC chan []byte, errC chan error) {
	for {
		r := bufio.NewReader(stdOut)
		// TODO: stop ignoring this isPrefix error-like bool
		line, _, err := r.ReadLine()
		if err != nil {
			errC <- err
			return
		}
		outputC <- line
		if !bytes.Contains(bytes.ToLower(line), bytes.ToLower(killOn)) {
			continue
		}
		outputC <- []byte("BabySitter: found kill_on string, sending kill signal")
		if err := cmd.Process.Kill(); err != nil {
			errC <- err
		}
		close(outputC)
		return
	}
}

var errGaveUsage = fmt.Errorf("gave usage, exiting")

func parseArgs() ([]byte, []string, error) {
	var killOn string
	var command []string
	for i := 0; i < len(os.Args); i++ {
		switch strings.ToLower(os.Args[i]) {
		case "-help", "-h":
			fmt.Println(`Usage: babysitter [options] -- command [args]
			-k, --kill_on <string>  String to kill on
			-h, --help              Show this help`)
			return nil, nil, errGaveUsage
		case "-kill_on", "-k":
			killOn = os.Args[i+1]
			i++
		case "--":
			command = os.Args[i+1:]
			return []byte(killOn), command, nil
		}
	}
	return nil, nil, fmt.Errorf("no command specified")
}
