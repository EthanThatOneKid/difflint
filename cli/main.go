// Run:
// git diff | go run cli/main.go

package main

import (
	"io"
	"log"
	"os"

	"github.com/ethanthatonekid/difflint"
	"github.com/urfave/cli/v2"
)

func main() {
	app := NewApp()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

type App struct {
	*cli.App
}

func NewApp() *App {
	app := &App{}

	app.App = &cli.App{
		Name:  "difflint",
		Usage: "lint diffs from standard input",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "include",
				Usage:    "include files matching the given glob",
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:     "exclude",
				Usage:    "exclude files matching the given glob",
				Required: false,
			},
			&cli.PathFlag{
				Name:     "ext_map",
				Usage:    "path to file extension map[string][]string (see README.md for format)",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "verbose",
				Usage:    "enable verbose logging",
				Required: false,
			},
		},
		Before: func(ctx *cli.Context) error {
			if ctx.Bool("verbose") {
				log.SetOutput(ctx.App.ErrWriter)
				log.SetFlags(log.Ltime)
			} else {
				log.SetOutput(io.Discard)
			}
			return nil
		},
		After: func(ctx *cli.Context) error {
			log.SetOutput(ctx.App.ErrWriter)
			return nil
		},
		Action: action,
	}

	return app
}

func action(ctx *cli.Context) error {
	include := ctx.StringSlice("include")
	exclude := ctx.StringSlice("exclude")
	extMapPath := ctx.String("ext_map")

	unsatisfiedRules, err := difflint.Do(ctx.App.Reader, include, exclude, extMapPath)
	if err != nil {
		return err
	}

	if len(unsatisfiedRules) > 0 {
		return cli.Exit(unsatisfiedRules.String(), 1)
	}

	return nil
}
