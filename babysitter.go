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
	if err != nil {
		log.Fatal(err)
	}

	if err := command.start(); err != nil {
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

func newWatchedCommand() (wc watchedCommand, err error) {
	var killOn []byte
	var cmdStrs []string
	for i := 0; i < len(os.Args); i++ {
		switch strings.ToLower(os.Args[i]) {
		case "-help", "-h":
			fmt.Println(`Usage: babysitter [options] -- command [args]
			-k, --kill_on <string>  String to kill on
			-h, --help              Show this help`)
			os.Exit(1)
		case "-kill_on", "-k":
			killOn = []byte(os.Args[i+1])
			i++
		case "--":
			cmdStrs = os.Args[i+1:]
		}
	}
	if len(killOn) == 0 {
		return wc, fmt.Errorf("no kill_on specified")
	}
	if len(cmdStrs) == 0 {
		return wc, fmt.Errorf("no command specified")
	}

	//nolint: gosec // this tool is run locally, not a security risk
	cmd := exec.Command(cmdStrs[0], cmdStrs[1:]...)
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	wc = watchedCommand{
		combOutC: make(chan []byte),
		errorC:   make(chan error),
		stdOut:   stdOut,
		stdErr:   stdErr,
		cmdStrs:  cmdStrs,
		cmd:      cmd,
		killOn:   killOn,
	}

	return
}

func (wc watchedCommand) start() error {
	return wc.cmd.Start()
}
