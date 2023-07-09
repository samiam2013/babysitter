package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	killOn, command, err := parseArgs()
	if err == errGaveUsage {
		return
	}
	if err != nil {
		log.Fatal(err)
	}

	// outputC is checked below for a close() signal
	//  to stop the utility when the subcommand finishes
	outputC := make(chan []byte)
	errorC := make(chan error)

	go babysit(command, []byte(killOn), outputC, errorC)

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
	//nolint: gosec // this tool is run locally, not a security risk
	cmd := exec.Command(command[0], command[1:]...)
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		errorC <- err
		return
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		errorC <- err
		return
	}
	if err := cmd.Start(); err != nil {
		errorC <- err
		return
	}
	go listenAndKill(cmd, killOn, stdOut, stdErr, outputC, errorC)
}

func listenAndKill(
	cmd *exec.Cmd, killOn []byte, stdOut, stdErr io.Reader,
	outputC chan []byte, errC chan error) {

	ctx, cancel := context.WithCancel(context.Background())
	stdOutBuf := bufio.NewReader(stdOut)
	stdErrBuf := bufio.NewReader(stdErr)

	go func() {
		for {
			readFrom(ctx, cancel, stdOutBuf, errC, outputC, cmd, killOn)
		}
	}()
	go func() {
		for {
			readFrom(ctx, cancel, stdErrBuf, errC, outputC, cmd, killOn)
		}
	}()

}

func readFrom(ctx context.Context, cancel context.CancelFunc,
	r *bufio.Reader, errC chan error, outputC chan []byte,
	cmd *exec.Cmd, killOn []byte) {
	select {
	case <-ctx.Done():
		return
	default:
		line, _, err := r.ReadLine()
		if err != nil {
			errC <- err
			return
		}

		outputC <- line

		if bytes.Contains(bytes.ToLower(line), bytes.ToLower(killOn)) {
			outputC <- []byte("BabySitter: found string, sending kill signal")
			if err := cmd.Process.Kill(); err != nil {
				errC <- err
			}
			cancel()
			close(outputC)
		}
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
	if len(killOn) == 0 {
		return nil, nil, fmt.Errorf("no kill_on specified")
	} else {
		fmt.Println("kill_on:", killOn)
	}
	return nil, nil, fmt.Errorf("no command specified")
}
