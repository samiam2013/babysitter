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
	var (
		killOn  string
		command []string
	)
	for i := 0; i < len(os.Args); i++ {
		switch strings.ToLower(os.Args[i]) {
		case "-help", "-h":
			help()
			return
		case "-kill_on", "-k":
			killOn = os.Args[i+1]
			i++
		case "--":
			command = os.Args[i+1:]
		}
		if len(command) > 0 {
			break
		}
	}
	fmt.Println("killOn:", killOn)
	fmt.Println("command:", command)
	if len(command) == 0 {
		log.Fatal("No command given")
	}
	// channel for getting back the output

	// start the babysat program
	outputC := make(chan []byte)
	errorC := make(chan error)
	go babysit(command, killOn, outputC, errorC)

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
			fmt.Printf("output: %s\n", string(output))
		}
	}
}

func babysit(command []string, killOn string, outputC chan []byte, errorC chan error) {
	fmt.Println("babysitting:", command)

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
	go func(stdOut io.ReadCloser, outputC chan []byte, errC chan error) {
		for {
			r := bufio.NewReader(stdOut)
			// TODO: why ignore this isPrefix?
			line, _, err := r.ReadLine()
			if err != nil {
				errC <- err
				return
			}
			fmt.Printf("line: %s\n", string(line))
			if !bytes.Contains(bytes.ToLower(line), bytes.ToLower([]byte(killOn))) {
				continue
			}
			outputC <- []byte("BabySitter: found killOn string, sending kill\n")
			if err := cmd.Process.Kill(); err != nil {
				errC <- err
			}
			close(outputC)
			return
		}
	}(stdOut, outputC, errorC)
}

func help() {
	fmt.Println(`Usage: babysitter [options] -- command [args]
	-k, --kill_on <string>  String to kill on
	-h, --help              Show this help`)
}
