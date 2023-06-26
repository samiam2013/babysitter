package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("Starting iteration")
		time.Sleep(time.Second * 1)
		fmt.Println("insert some api call here")
		time.Sleep(time.Second * 1)
		fmt.Println("insert crazy amounts of crunch here")
		time.Sleep(time.Second * 1)
		fmt.Println("Successfully complete stop finished iteration " +
			"sleeping for 3 seconds")
	}
}
