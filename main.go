package main

import (
	"log"
	"playlist/cli"
)

func main() {
	// temp run GUI here
	/*if err := gui.Run(); err != nil {
		log.Fatal(err)
	}*/

	if err := cli.Run(); err != nil {
		log.Fatal(err)
	}
}
