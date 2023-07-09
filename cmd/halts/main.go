package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	for {
		fmt.Println("Starting iteration")
		time.Sleep(time.Second * 1)
		fmt.Println("insert some api call here")
		time.Sleep(time.Second * 1)
		fmt.Println("insert crazy amounts of crunch here")
		if _, err := fmt.Fprintln(os.Stderr, "error channel output here"); err != nil {
			fmt.Println("Error writing to stderr")
		}
		time.Sleep(time.Second * 1)
		fmt.Println("Successfully complete stop finished iteration " +
			"sleeping for 3 seconds")
	}
}
