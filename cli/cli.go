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
	Commands:    []*cli.Command{get, list},
}

var get = &cli.Command{
	Name:        "get",
	Description: "",
	Usage:       "",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "do not print info to stderr"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		args := c.Args()

		if !args.Present() {
			return fmt.Errorf("need playlist url arg")
		}
		if err := pls.ValidatePlaylistUrl(args.First()); err != nil {
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
	Usage:       "",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "query", Aliases: []string{"q"}, Usage: "fuzzy search query"},
		&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Usage: "number of results. Must be a positive integer", Action: NotNegative},
		&cli.BoolFlag{Name: "tui", Aliases: []string{"t"}, Usage: "launch a tui and browse the result instead of printing it to stdout"},
		// the tui does not let you filter or search or anything like that (you do that with the cli flags above)
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
			return pls.TUI(videos)
		}

		// TODO print URLs???

		for i, video := range videos {
			fmt.Printf("%d: %s\n", i+1, video.Title)
		}

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
