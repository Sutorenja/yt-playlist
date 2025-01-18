package pls

import (
	"encoding/json"
	"os"
	"os/exec"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Channel struct {
	// the name of the channel that uploaded the video
	// e.g. Alydle
	ChannelTitle string `json:"channel"` // "channel" is just the display name

	// the follower count of the channel that uploaded the video
	ChannelFollowerCount int `json:"channel_follower_count"`
	// is null in structs...
	// does that just set it to 0? think so
	// TODO test it

	// the id of the channel that uploaded the video
	// e.g. UCueMs7IubhfQm3jXMc0NYOw
	ChannelId string `json:"channel_id"`

	// url of the channel
	// e.g. https://www.youtube.com/channel/UCueMs7IubhfQm3jXMc0NYOw
	ChannelUrl string `json:"channel_url"`

	// e.g. @Yehyeobbun
	UploaderId string `json:"uploader_id"`
}

type Video struct {
	gorm.Model

	// channel that uploaded the video
	Channel

	// video id
	// e.g. rlHBvH87G14
	Id string `json:"id" gorm:"primaryKey"` // TODO can a video be present multiple times in the playlist??? if so....  WAIT NO it cant.... phew

	// video title
	// e.g. 'Are We Living in the Gooner Gacha Age?'
	Title string `json:"title"`

	// video thumbnail
	// e.g. 'https://i.ytimg.com/vi/rlHBvH87G14/maxresdefault.jpg'
	Thumbnail string `json:"thumbnail"`

	// video description
	// can be of arbitrary length and contains line breaks
	Description string `json:"description"`

	// video view count
	// e.g. 258948
	ViewCount int `json:"view_count"`

	// amount of comments on the video
	// e.g. 405
	CommentCount int `json:"comment_count"`

	// amount of likes on the video
	// e.g. 15847
	LikeCount int `json:"like_count"`

	// video url
	// e.g. https://www.youtube.com/watch?v=rlHBvH87G14
	VideoUrl string `json:"webpage_url"`

	// video upload date
	// e.g. 20200719
	UploadDate string `json:"upload_date"`

	// the exact timestamp the video was uploaded
	// e.g. 1595133655
	CreationTimestamp int `json:"timestamp"`

	// video availability
	// can be either "public", "private", or "unlisted"
	Availability string `json:"availability"`

	// video categories
	// e.g. ["Entertainment", "Gaming"]
	Categories []string `json:"categories"`

	// user defined video tags
	// its just a list of user defined strings
	Tags []string `json:"tags"`

	// video length
	// e.g. 447
	Duration int `json:"duration"`

	// video length as a human-readable string
	// e.g. "7:27"
	DurationString string `json:"duration_string"`

	// the index of the video when in a playlist
	// this field is only present if the video was
	// unmarshalled from a playlist url
	PlaylistIndex int `json:"playlist_index,omitempty"`
}

type Playlist struct {
	// channel that created the playlist
	Channel

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

	// list of user defined tags
	Tags []string `json:"tags"`

	// number of videos in the playlist
	Count int `json:"playlist_count"`

	// last time the playlist was modified
	// e.g. 20231125
	ModifiedDate string `json:"modified_date"`

	// list of the videos in the playlist
	Entries []Video `json:"entries"`
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
	// TODO sanitate playlistUrl
	// make sure its actually a valid url
	// by running it through net/url?

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
