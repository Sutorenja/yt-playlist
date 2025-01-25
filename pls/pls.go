package pls

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"sort"
	"time"

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
	VideoId string `json:"id" gorm:"primaryKey"`

	// video title
	// e.g. 'Are We Living in the Gooner Gacha Age?'
	Title string `json:"title"`

	// video description
	// can be of arbitrary length and contains line breaks
	Description string `json:"description"`

	// list of thumbnails in different resolutions
	Thumbnails []Thumbnail `json:"thumbnails" gorm:"serializer:json"`

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

func FuzzyFind(query string, videos []Video) []Video {
	var words []string
	var word2video = make(map[string]Video)
	for _, video := range videos {
		word := fmt.Sprintf("%s %s", video.ChannelTitle, video.Title)
		word2video[word] = video
		words = append(words, word)
	}

	matches := fuzzy.RankFindNormalizedFold(query, words)
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

func GetAllVideos(db *gorm.DB) ([]Video, error) {
	// we can order by gorm.Model's CreatedAt (or ID alternatively)
	// to get the "youtube" custom ordering (i.e. "sort by manual")
	// use db.Order("created_at DESC").Find(&videos)
	// actually i think that is the default order... so we dont have to do anything
	var videos []Video
	res := db.Find(&videos)
	return videos, res.Error
}

func (video Video) BiggestThumbnail() Thumbnail {
	biggest := Thumbnail{"", 0, 0}
	for _, thumbnail := range video.Thumbnails {
		area := thumbnail.Width * thumbnail.Height
		areaBiggest := biggest.Width * biggest.Height

		if area > areaBiggest {
			biggest = thumbnail
		}
	}
	return biggest
}

func (channel Channel) Url() string {
	if channel.UploaderId != "" {
		return "youtube.com/" + channel.UploaderId
	}
	return "youtube.com/channel/" + channel.ChannelId
}

func (video Video) Url() string {
	return "youtube.com/watch?v=" + video.VideoId
}

// video length as a human-readable string
// format: days:hours:minutes:seconds
// days is not fixed length
// all other are always 2 digits long
// e.g. 1:00:00:00
func (video Video) DurationString() string {
	duration := time.Duration(video.Duration) * time.Second

	hours := duration.Truncate(time.Hour)
	duration = duration - hours

	min := duration.Truncate(time.Minute)
	duration = duration - min

	days := int(hours.Hours() / 24)
	hrs := int(hours.Hours() - float64(days*24))

	str := fmt.Sprintf("%02d:%02d", int(min.Minutes()), int(duration.Seconds()))
	if days > 0 {
		return fmt.Sprintf("%d:%02d:%s", days, hrs, str)
	}
	if hrs > 0 {
		return fmt.Sprintf("%02d:%s", hrs, str)
	}
	return str
}
