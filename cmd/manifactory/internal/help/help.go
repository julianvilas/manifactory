package help

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/julianvilas/manifactory/cmd/manifactory/internal/base"
)

func Help(args []string) {
	if len(args) == 0 {
		PrintUsage()
		return
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: manifactory help command\n\nToo many arguments given.\n")
		os.Exit(2)
	}

	arg := args[0]

	for _, cmd := range base.Commands {
		if cmd.Name() == arg {
			tmpl(helpTemplate, cmd)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown help topic: %#q. Run 'manifactory help'.\n", arg)
	os.Exit(2)
}

func PrintUsage() {
	tmpl(usageTemplate, base.Commands)
}

func tmpl(text string, data interface{}) {
	t := template.New("top")
	t.Funcs(template.FuncMap{"trim": strings.TrimSpace})
	template.Must(t.Parse(text))
	if err := t.Execute(os.Stderr, data); err != nil {
		panic(err)
	}
}

const usageTemplate = `manifactory is a tool to interact with a JFrog Artifactory Container Registry

Usage:

	manifactory command [arguments]

The commands are:
{{range .}}
	{{.Name | printf "%-11s"}} {{.Short}}{{end}}

Use "manifactory help [command]" for more information about a command.
`

const helpTemplate = `usage: manifactory {{.UsageLine}}

{{.Long | trim}}
`
