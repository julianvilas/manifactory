package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/julianvilas/manifactory/cmd/manifactory/internal/base"
	"github.com/julianvilas/manifactory/cmd/manifactory/internal/help"
	"github.com/julianvilas/manifactory/cmd/manifactory/internal/repos"
	"github.com/julianvilas/manifactory/cmd/manifactory/internal/tags"
)

func init() {
	base.Commands = []*base.Command{
		repos.CmdRepos,
		tags.CmdTags,
	}

	base.Usage = mainUsage
}

func main() {
	log.SetFlags(0)

	flag.Usage = base.Usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		base.Usage()
		os.Exit(2)
	}

	if args[0] == "help" {
		help.Help(args[1:])
		return
	}

	for _, cmd := range base.Commands {
		cmd.Flag.Usage = cmd.Usage
		if cmd.Name() == args[0] {
			cmd.Flag.Parse(args[1:])
			args = cmd.Flag.Args()
			if err := cmd.Run(cmd, args); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	fmt.Fprintf(os.Stderr, "manifactory: unknown subcommand %q\nRun 'manifactory help' for usage.\n", args[0])
	os.Exit(2)
}

func mainUsage() {
	help.PrintUsage()
	os.Exit(2)
}
