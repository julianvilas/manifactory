package tags

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/julianvilas/manifactory/cmd/manifactory/internal/base"
	"github.com/julianvilas/manifactory/internal/client"
)

var CmdTags = &base.Command{
	UsageLine: "tags [-u string] [-p string] [-ibn] <registry-url> [repos-file]",
	Short:     "lists tags for a list of repos in a container registry",
	Long: `
Prints to the standard output all the existing tags in a JFrog Artifactory
Container Registry, given a list of repositories.

The list of repositories format should match the output format of the 'repos'
command. It must be provided via stdin or by indicating a [repos-file].

The default output is a one-per-line 'repository/tag' collection. The -n flag
turns the output be 'registry-url/repository[: | @]tag' instead (':' for image
tags, '@' for digests).
	`,
}

var (
	userFlag      string
	passFlag      string
	insecureFlag  bool
	basicAuthFlag bool
	namesFlag     bool
)

func init() {
	CmdTags.Run = runTags
	CmdTags.Flag.StringVar(&userFlag, "u", "", "registry username")
	CmdTags.Flag.StringVar(&passFlag, "p", "", "registry password")
	CmdTags.Flag.BoolVar(&insecureFlag, "i", false, "skip registry SSL/TLS validations")
	CmdTags.Flag.BoolVar(&basicAuthFlag, "b", false, "use Basic auth instead of Bearer tokens")
	CmdTags.Flag.BoolVar(&namesFlag, "n", false, "print pullable image names")
}

func runTags(cmd *base.Command, args []string) error {
	var reader io.Reader
	switch n := len(args); {
	case n < 1:
		return errors.New("<registry-url> argument required")
	case n == 1:
		reader = os.Stdin
		log.Println("reading from stdin")
	case n == 2:
		f, err := os.Open(args[1])
		if err != nil {
			return err
		}
		reader = f
	case n > 2:
		return errors.New("too many arguments")
	}

	reg := args[0]
	regURL, err := url.Parse(reg)
	if err != nil {
		return fmt.Errorf("incorrect registry URL: %w", err)
	}

	opts := client.Options{
		Insecure:  insecureFlag,
		BasicAuth: basicAuthFlag,
	}
	cli := client.New(reg, userFlag, passFlag, opts)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		repo := scanner.Text()
		tags, err := cli.Tags(repo)
		if err != nil {
			log.Printf("can not get tags from %s: %s", repo, err)
		}
		for _, tag := range tags {
			if namesFlag {
				sep := ":"
				if strings.HasPrefix(tag, "sha256:") {
					sep = "@"
				}

				image, err := url.JoinPath(regURL.Host, repo)
				if err != nil {
					return err
				}

				fmt.Printf("%s%s%s\n", image, sep, tag)
			} else {
				fmt.Printf("%s/%s\n", repo, tag)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("can not read the input file: %w", err)
	}

	return nil
}
