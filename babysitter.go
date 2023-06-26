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
		log.Fatal(err)
	}
	// channels for getting back the output or err
	outputC := make(chan []byte) // closed for internal "done" signaling
	errorC := make(chan error)

	// start the babysat program
	go babysit(command, []byte(killOn), outputC, errorC)

	// forever wait for a string to be handed back or a timeout
	for {
		select {
		case err := <-errorC:
			fmt.Printf("error: %s\n", err)
		case output, open := <-outputC:
			if !open {
				fmt.Println("outputC closed")
				return
			}
			fmt.Printf("output: %s", string(output))
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
		outputC <- []byte("BabySitter: found killOn string, sending kill")
		if err := cmd.Process.Kill(); err != nil {
			errC <- err
		}
		close(outputC)
		return
	}
}

func parseArgs() ([]byte, []string, error) {
	var killOn string
	var command []string
	for i := 0; i < len(os.Args); i++ {
		switch strings.ToLower(os.Args[i]) {
		case "-help", "-h":
			usage()
			return nil, nil, nil
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

func usage() {
	fmt.Println(`Usage: babysitter [options] -- command [args]
	-k, --kill_on <string>  String to kill on
	-h, --help              Show this help`)
}
