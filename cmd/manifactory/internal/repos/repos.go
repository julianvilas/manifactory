package repos

import (
	"errors"
	"fmt"

	"github.com/julianvilas/manifactory/cmd/manifactory/internal/base"
	"github.com/julianvilas/manifactory/internal/client"
)

var CmdRepos = &base.Command{
	UsageLine: "repos [-u string] [-p string] [-ib] <registry-url>",
	Short:     "lists repositories in a container registry",
	Long: `
Prints to the standard output all the repositories available in a JFrog
Artifactory Container Registry listening at <registry-url>.
	`,
}

var (
	userFlag      string
	passFlag      string
	insecureFlag  bool
	basicAuthFlag bool
)

func init() {
	CmdRepos.Run = runRepos
	CmdRepos.Flag.StringVar(&userFlag, "u", "", "registry username")
	CmdRepos.Flag.StringVar(&passFlag, "p", "", "registry password")
	CmdRepos.Flag.BoolVar(&insecureFlag, "i", false, "skip registry SSL/TLS validations")
	CmdRepos.Flag.BoolVar(&basicAuthFlag, "b", false, "use Basic auth instead of Bearer tokens")
}

func runRepos(cmd *base.Command, args []string) error {
	switch n := len(args); {
	case n < 1:
		return errors.New("<registry-url> argument required")
	case n > 1:
		return errors.New("too many arguments")
	}

	opts := client.Options{
		Insecure:  insecureFlag,
		BasicAuth: basicAuthFlag,
	}
	cli := client.New(args[0], userFlag, passFlag, opts)

	catalog, err := cli.Catalog()
	if err != nil {
		return fmt.Errorf("can not get catalog: %w", err)
	}
	for _, repo := range catalog {
		fmt.Println(repo)
	}

	return nil
}
