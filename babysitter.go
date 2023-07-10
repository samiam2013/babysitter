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

type watchedCommand struct {
	cmdStrs  []string
	cmd      *exec.Cmd
	killOn   []byte
	stdOut   io.Reader
	stdErr   io.Reader
	combOutC chan []byte
	errorC   chan error
}

func main() {
	command, err := newWatchedCommand()
	if err == errGaveUsage {
		return
	}
	if err != nil {
		log.Fatal(err)
	}

	go listenAndKill(command)

	for {
		select {
		case err := <-command.errorC:
			log.Fatal("Error from command: ", err)
		case output, open := <-command.combOutC:
			if !open {
				log.Println("Output closed")
				return
			}
			log.Printf("Output: %s\n", string(output))
		}
	}
}

func listenAndKill(command watchedCommand) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			readFrom(ctx, cancel, bufio.NewReader(command.stdOut), command)
		}
	}()
	go func() {
		for {
			readFrom(ctx, cancel, bufio.NewReader(command.stdErr), command)
		}
	}()
}

// listens to the context waiting for the other copy reading the other pipe
// to cancel the context, otherwise reads from the pipe and sends the output
func readFrom(ctx context.Context, cancel context.CancelFunc, r *bufio.Reader,
	command watchedCommand) {
	select {
	case <-ctx.Done():
		return
	default:
		line, _, err := r.ReadLine()
		if err != nil {
			command.errorC <- err
			return
		}

		command.combOutC <- line

		if bytes.Contains(bytes.ToLower(line), bytes.ToLower(command.killOn)) {
			command.combOutC <- []byte("Found string, sending kill signal")
			if err := command.cmd.Process.Kill(); err != nil {
				command.errorC <- err
			}
			cancel()
			close(command.combOutC)
		}
	}
}

var errGaveUsage = fmt.Errorf("gave usage, exiting")

func newWatchedCommand() (watchedCommand, error) {
	wc := watchedCommand{
		combOutC: make(chan []byte),
		errorC:   make(chan error),
	}

	for i := 0; i < len(os.Args); i++ {
		switch strings.ToLower(os.Args[i]) {
		case "-help", "-h":
			fmt.Println(`Usage: babysitter [options] -- command [args]
			-k, --kill_on <string>  String to kill on
			-h, --help              Show this help`)
			return wc, errGaveUsage
		case "-kill_on", "-k":
			wc.killOn = []byte(os.Args[i+1])
			i++
		case "--":
			wc.cmdStrs = os.Args[i+1:]
		}
	}
	if len(wc.killOn) == 0 {
		return wc, fmt.Errorf("no kill_on specified")
	}
	if len(wc.cmdStrs) == 0 {
		return wc, fmt.Errorf("no command specified")
	}

	//nolint: gosec // this tool is run locally, not a security risk
	wc.cmd = exec.Command(wc.cmdStrs[0], wc.cmdStrs[1:]...)
	stdOut, err := wc.cmd.StdoutPipe()
	if err != nil {
		return wc, err
	}
	stdErr, err := wc.cmd.StderrPipe()
	if err != nil {
		return wc, err
	}
	wc.stdOut = stdOut
	wc.stdErr = stdErr
	if err := wc.cmd.Start(); err != nil {
		return wc, err
	}

	return wc, nil
}
