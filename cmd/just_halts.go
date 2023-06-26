package main

import (
	"time"
	"fmt"
)


func main() {
	for {
		time.Sleep(time.Second * 3)
		fmt.Println("Successfully complete stop finished iteration "+
			"sleeping for 3 seconds")
	}
}

