package pls

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os/exec"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Thumbnail struct {
	Url    string
	Height int
	Width  int
}

type Channel struct {
	// channel display name
	// e.g. Mudan
	ChannelTitle string `json:"channel"`

	// channel id
	// e.g. UCZTgg6AiQkSHtL5Jj0IO6MQ
	ChannelId string `json:"channel_id"`

	// channel id (new system)
	// e.g. @Mudan
	UploaderId string `json:"uploader_id"`
}

type Video struct {
	gorm.Model

	// channel that uploaded the video
	Channel

	// video id
	// e.g. rlHBvH87G14
	Id string `json:"id" gorm:"primaryKey"`

	// video title
	// e.g. 'Are We Living in the Gooner Gacha Age?'
	Title string `json:"title"`

	// video description
	// can be of arbitrary length and contains line breaks
	Description string `json:"description"`

	// list of thumbnails in different resolutions
	Thumbnails []Thumbnail `json:"thumbnails"`

	// video view count
	// e.g. 258948
	ViewCount int `json:"view_count"`

	// video length
	// e.g. 447
	Duration float64 `json:"duration"`
}

type Playlist struct {
	// playlist id
	// e.g. PLA9DML3OBu8nAICrUUCYTkELNoyMPzv2m
	Id string `json:"id"`

	// playlist title
	// e.g. (G)I-DLE
	Title string `json:"title"`

	// video availability
	// can be either "public", "private", or "unlisted"
	Availability string `json:"availability"`

	// playlist description
	// can be of arbitrary length and contains line breaks
	Description string `json:"description"`

	// number of videos in the playlist
	Count int `json:"playlist_count"`

	// last time the playlist was modified
	// e.g. 20231125
	ModifiedDate string `json:"modified_date"`

	// list of the videos in the playlist
	Entries []Video `json:"entries"`
}

// simple func that does all the DB stuff
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
func DownloadPlaylist(playlistUrl string, logger io.Writer) (*Playlist, error) {
	err := ValidatePlaylistUrl(playlistUrl)
	if err != nil {
		return nil, err
	}

	// hardcoding firefox here is kinda bad, but i dont feel like changing it rn
	args := []string{playlistUrl, "-J", "--verbose", "--flat-playlist", "--ignore-no-formats-error", "--cookies-from-browser", "firefox"}
	ytdlp := exec.Command("yt-dlp", args...)
	ytdlp.Stderr = logger

	stdout, err := ytdlp.Output()
	if err != nil {
		return nil, err
	}
	return UnmarshalPlaylist(stdout)
}

func UnmarshalPlaylist(data []byte) (*Playlist, error) {
	var playlist Playlist
	err := json.Unmarshal(data, &playlist)
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

func ValidatePlaylistUrl(rawUrl string) error {
	url, err := url.Parse(rawUrl)
	if err != nil {
		return err
	}
	if url.Scheme != "https" {
		return errors.New("expected https")
	}
	if url.Hostname() != "youtube.com" && url.Hostname() != "www.youtube.com" {
		return errors.New("not a youtube url")
	}
	if url.Path != "/playlist" {
		return errors.New("not a playlist url")
	}
	if !url.Query().Has("list") {
		return errors.New("no playlist id in url")
	}
	return nil
}

func FuzzyFind(query string, filter string, videos []Video) []Video {
	var words []string
	var word2video = make(map[string]Video)
	query = strings.ToLower(query)

	for _, video := range videos {
		// TODO proper error handling
		word := ""
		switch filter {
		case "title":
			word = strings.ToLower(video.Title)
		case "desc":
			word = strings.ToLower(video.Description)
		case "channel":
			word = strings.ToLower(video.ChannelTitle)
		}
		word2video[word] = video
		words = append(words, word)
	}

	matches := fuzzy.RankFind(query, words)
	sort.Sort(matches)

	videos = []Video{}
	for _, v := range matches {
		video, ok := word2video[v.Target]
		if !ok {
			continue
		}
		videos = append(videos, video)
	}
	return videos
}
