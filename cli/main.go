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
	extMapPath := ctx.String("extmap")

	if err := difflint.Do(ctx.App.Reader, include, exclude, &extMapPath); err != nil {
		return err
	}

	return nil
}
