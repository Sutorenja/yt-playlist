package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Video struct {
	gorm.Model
	Id                   string `json:"id" gorm:"primaryKey"`
	Title                string `json:"title"`
	Thumbnail            string `json:"thumbnail"`
	Description          string `json:"description"`
	ViewCount            int    `json:"view_count"`
	CommentCount         int    `json:"comment_count"`
	LikeCount            int    `json:"like_count"`
	VideoUrl             string `json:"webpage_url"`
	CreationTimestamp    int    `json:"timestamp"` // This is the exact timestamp the video was uploaded
	ChannelTitle         string `json:"channel"`   // "channel" is just the display name
	ChannelFollowerCount int    `json:"channel_follower_count"`
	ChannelId            string `json:"channel_id"`
	ChannelUrl           string `json:"channel_url"`
	UploaderId           string `json:"uploader_id"`
	Availability         string `json:"availability"` // "public", "private", or "unlisted" etc.
}

/*
	"like_count": 15847,
	"channel": "Alydle",
	"channel_follower_count": 71800,
	"uploader": "Alydle",
	"uploader_id": "@Yehyeobbunn",
	"uploader_url": "https://www.youtube.com/@Yehyeobbunn",
	"upload_date": "20200719",
	"timestamp": 1595133655,
	"availability": "public",
	"original_url": "https://www.youtube.com/watch?v=rlHBvH87G14",
	"webpage_url_basename": "watch",
	"webpage_url_domain": "youtube.com",
	"extractor": "youtube",
	"extractor_key": "Youtube",
	"playlist_count": 5,
	"playlist": "(G)I-DLE",
	"playlist_id": "PLA9DML3OBu8nAICrUUCYTkELNoyMPzv2m",
	"playlist_title": "(G)I-DLE",
	"playlist_uploader": "sutoremnja",
	"playlist_uploader_id": "@sutorenjaa",
	"playlist_channel": "sutoremnja",
	"playlist_channel_id": "UCm2XsKMy-20_IVmSX9iSqEA",
	"n_entries": 5,
	"playlist_index": 1,
	"__last_playlist_index": 5,
	"playlist_autonumber": 1,
	"display_id": "rlHBvH87G14",
	"fulltitle": "(G)I-DLE making me question their sanity",
	"duration_string": "7:27",
*/

type Playlist struct {
	Id           string  `json:"id"`
	Title        string  `json:"title"`
	Availability string  `json:"availability"`
	Count        int     `json:"playlist_count"`
	ModifiedDate string  `json:"modified_date"`
	Entries      []Video `json:"entries"`

	/* just info about the playlist author
	"channel": "sutoremnja"
	"channel_id": "UCm2XsKMy-20_IVmSX9iSqEA"
	"uploader_id": "@sutorenjaa"
	"uploader": "sutoremnja"
	"channel_url": "https://www.youtube.com/channel/UCm2XsKMy-20_IVmSX9iSqEA"
	"uploader_url": "https://www.youtube.com/@sutorenjaa"*/
}

func main() {
	// works and does not install videos, only metadata
	//  yt-dlp https://www.youtube.com/playlist?list=PLA9DML3OBu8nAICrUUCYTkELNoyMPzv2m -J | jq . > out2.json

	// https://www.youtube.com/playlist?list=WL
	// https://www.youtube.com/playlist?list=PLA9DML3OBu8nAICrUUCYTkELNoyMPzv2m

	playlist, err := DownloadPlaylist("https://www.youtube.com/playlist?list=PLA9DML3OBu8nAICrUUCYTkELNoyMPzv2m")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("playlist.Title: %v\n", playlist.Title)
	fmt.Printf("playlist.Count: %v\n", playlist.Count)

	db, err := DB(playlist.Title + ".sqlite")
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range playlist.Entries {
		fmt.Printf("v.Title: %v\n", v.Title)
		db.Create(&v)
	}

}

// just create a func that does all the DB stuff
// and we can just call that func (it returns DB and err)
// and check for err rq, then we have a DB we can use!
// its a bit boilerplate-y but its fine
func DB(fn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(fn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Video{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// https://www.reddit.com/r/youtubedl/wiki/cookies
func DownloadPlaylist(playlistUrl string) (*Playlist, error) {
	// TODO
	// launch firefox with a youtube tab open?
	// (you can close it later)

	cmd := exec.Command("yt-dlp", playlistUrl, "-J" /*, "--cookies-from-browser", "firefox"*/) // TODO cookies
	cmd.Stderr = os.Stderr

	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var playlist Playlist

	err = json.Unmarshal(stdout, &playlist)
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}
