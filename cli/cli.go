package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"playlist/pls"

	"github.com/urfave/cli/v3"
)

func Run() error {
	return root.Run(context.Background(), os.Args)
}

var root = &cli.Command{
	Description: "",
	Usage:       "",
	Commands:    []*cli.Command{get, list, has},
}

var get = &cli.Command{
	Name:        "get",
	Description: "",
	Usage:       "pls get [youtube playlist url] [flags...]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "do not print info to stderr"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		args := c.Args()

		if !args.Present() {
			return fmt.Errorf("need playlist url arg")
		}
		if err := pls.ValidatePlaylistFeedUrl(args.First()); err != nil {
			return fmt.Errorf("not a valid playlist url: %s", err)
		}

		var writer io.Writer = os.Stderr
		if c.Bool("quiet") {
			writer = io.Discard
		}
		writer.Write([]byte(""))

		playlist, err := pls.DownloadPlaylist(args.First(), writer)
		if err != nil {
			return err
		}
		db, err := pls.DB(playlist.Title + ".sqlite")
		if err != nil {
			return err
		}
		for _, v := range playlist.Entries {
			db.Create(&v)
		}
		writer.Write([]byte("sqlite database created\n"))
		return nil
	},
}

var list = &cli.Command{
	Name:        "list",
	Description: "",
	Usage:       "pls list [sqlite file] [flags...]",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "query", Aliases: []string{"q"}, Usage: "fuzzy search query"},
		&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Usage: "number of results. Must be a positive integer", Action: NotNegative},
		&cli.BoolFlag{Name: "tui", Aliases: []string{"t"}, Usage: "launch a tui and browse the result instead of printing it to stdout"},
		// the tui does not let you filter or search or anything like that (you do that with the cli flags above)

		// TODO add flags that let you specify output format. Instead of printing URL, we could print video title, channel title etc.
		// in the mean time, just use a url flag
		&cli.BoolFlag{Name: "url", Usage: "print video url"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		args := c.Args()

		if !args.Present() {
			return fmt.Errorf("need sqlite database file arg")
		}

		// so a problem here is that we can provide literally ANY filename we want afaik
		// and gorm/sqlite would probably allow it...

		db, err := pls.DB(args.First())
		if err != nil {
			return err
		}

		videos, err := pls.GetAllVideos(db)
		if err != nil {
			return err
		}

		if c.IsSet("query") {
			videos = pls.FuzzyFind(c.String("query"), videos)
		}

		n := len(videos)
		if c.IsSet("limit") {
			n = int(c.Int("limit"))
		}
		videos = videos[0:min(n, len(videos))]

		if c.Bool("tui") {
			videos, err = pls.TUI(videos)
			if err != nil {
				return err
			}
		}

		for i, video := range videos {
			if c.Bool("url") {
				fmt.Println(video.Url())
				continue
			}
			fmt.Printf("%d: %s\n", i+1, video.Title)
		}
		return nil
	},
}

var has = &cli.Command{
	Name:        "has",
	Description: "",
	Usage:       "pls has [sqlite file] [youtube url]",
	Action: func(ctx context.Context, c *cli.Command) error {
		args := c.Args()

		if args.Len() > 2 {
			return fmt.Errorf("need sqlite database file arg and url arg")
		}
		if err := pls.ValidateVideoUrl(args.Get(1)); err != nil {
			return err
		}
		id, err := pls.VideoUrlId(args.Get(1))
		if err != nil {
			return err
		}
		db, err := pls.DB(args.First())
		if err != nil {
			return err
		}
		videos, err := pls.GetAllVideos(db)
		if err != nil {
			return err
		}
		for _, video := range videos {
			if video.VideoId == id {
				fmt.Println("true")
				return nil
			}
		}
		fmt.Println("false")
		return nil
	},
}

// make sure int flag is not negative
func NotNegative(_ context.Context, _ *cli.Command, v int64) error {
	if v < 0 {
		return fmt.Errorf("Flag cannot be negative")
	}
	return nil
}
