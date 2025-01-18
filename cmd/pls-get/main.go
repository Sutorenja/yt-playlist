package main

import (
	"flag"
	"fmt"
	"log"
	"playlist/pls"
)

/*
> go run
go: no go files listed

> go run a
package a is not in std (C:\Users\Mathias\scoop\apps\go\current\src\a)

> go run .
runs perfectly fine

> go run . a b fgojfsdlf g
runs perfectly fine (just ignores the other args)
*/

func main() {
	// works and does not install videos, only metadata
	//  yt-dlp https://www.youtube.com/playlist?list=PLA9DML3OBu8nAICrUUCYTkELNoyMPzv2m -J | jq . > out2.json

	// https://www.youtube.com/playlist?list=WL
	// https://www.youtube.com/playlist?list=PLA9DML3OBu8nAICrUUCYTkELNoyMPzv2m

	// TODO define flags here...

	flag.Parse()

	playlistFile := flag.Arg(0)
	if playlistFile == "" {
		log.Fatal("no playlist file listed")
	}

	playlist, err := pls.DownloadPlaylist(playlistFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("playlist.Title: %v\n", playlist.Title)
	fmt.Printf("playlist.Count: %v\n", playlist.Count)

	db, err := pls.DB(playlist.Title + ".sqlite")
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range playlist.Entries {
		fmt.Printf("v.Title: %v\n", v.Title)
		db.Create(&v)
	}

	// TODO print "sqlite file created" or smth
}
