package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"playlist/pls"

	"github.com/urfave/cli/v3"
)

var (
	quietFlag  = &cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "do not print info to stderr"}
	limitFlag  = &cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Usage: "number of results. Must be a positive integer", Action: NotNegative}
	formatFlag = &cli.StringFlag{Name: "format", Aliases: []string{"f"}, Usage: "format output", Value: "{Index}: {Title}"}
	queryFlag  = &cli.StringFlag{Name: "query", Aliases: []string{"q"}, Usage: "automatically insert query and return results. If not set, opens fzf and you can manually search and select"}
)

func Run() error {
	return root.Run(context.Background(), os.Args)
}

var root = &cli.Command{
	Name:        "pls",
	Description: "",
	Usage:       "",
	Commands:    []*cli.Command{get, list, find, has},
}

var get = &cli.Command{
	Name:        "get",
	Description: "",
	Usage:       "pls get [youtube playlist url] [flags...]",
	Flags:       []cli.Flag{quietFlag},
	Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
		if !c.Args().Present() {
			return ctx, fmt.Errorf("need playlist url arg")
		}
		return ctx, nil
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		url := c.Args().First()
		if err := pls.ValidatePlaylistFeedUrl(url); err != nil {
			return fmt.Errorf("not a valid playlist url: %s", err)
		}

		var writer io.Writer = os.Stderr
		if c.Bool("quiet") {
			writer = io.Discard
		}
		writer.Write([]byte(""))

		playlist, err := pls.DownloadPlaylist(url, writer)
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
	Flags:       []cli.Flag{limitFlag, formatFlag},
	Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
		if !c.Args().Present() {
			return ctx, fmt.Errorf("need sqlite database file arg")
		}
		return ctx, nil
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		// so a problem here is that we can provide literally ANY filename we want afaik
		// and gorm/sqlite would probably allow it...

		db, err := pls.DB(c.Args().First())
		if err != nil {
			return err
		}

		videos, err := pls.GetAllVideos(db)
		if err != nil {
			return err
		}

		n := len(videos)
		if c.IsSet("limit") {
			n = int(c.Int("limit"))
		}
		videos = videos[0:min(n, len(videos))]
		return pls.PrintVideosWithFormat(c.String("format"), videos)
	},
}

var has = &cli.Command{
	Name:        "has",
	Description: "",
	Usage:       "pls has [sqlite file] [youtube url]",
	Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
		if c.Args().Len() > 2 {
			return ctx, fmt.Errorf("need sqlite database file arg and url arg")
		}
		return ctx, nil
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		args := c.Args()
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

var find = &cli.Command{
	Name:        "find",
	Description: "",
	Usage:       "pls find [sqlite file] [flags...]",
	Flags:       []cli.Flag{queryFlag, limitFlag, formatFlag},
	Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
		if !c.Args().Present() {
			return ctx, fmt.Errorf("need sqlite database file arg")
		}
		return ctx, nil
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		// so a problem here is that we can provide literally ANY filename we want afaik
		// and gorm/sqlite would probably allow it...

		db, err := pls.DB(c.Args().First())
		if err != nil {
			return err
		}

		videos, err := pls.GetAllVideos(db)
		if err != nil {
			return err
		}

		videos, err = pls.FuzzyFind(c.String("query"), videos)
		if err != nil {
			return err
		}

		n := len(videos)
		if c.IsSet("limit") {
			n = int(c.Int("limit"))
		}
		videos = videos[0:min(n, len(videos))]
		return pls.PrintVideosWithFormat(c.String("format"), videos)
	},
}

// make sure int flag is not negative
func NotNegative(_ context.Context, _ *cli.Command, v int64) error {
	if v < 0 {
		return fmt.Errorf("Flag cannot be negative")
	}
	return nil
}
