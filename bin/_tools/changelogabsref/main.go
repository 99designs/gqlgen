// Command changelogabsref rewrites the changelog section of a GoReleaser
// release body so that its pull request references are absolute.
//
// GoReleaser writes changelog entries as "* <sha> <message> (#123)". Pasted
// into a pull request in a different repository, GitHub would link "#123" to
// that repository's own pull request 123. This program drops the commit SHA
// and rewrites each "#123" as "[owner/repo#123](https://github.com/owner/repo/pull/123)".
//
// Usage:
//
//	gh release view -R vektah/gqlparser --json body --jq .body |
//		go run ./changelogabsref https://github.com/vektah/gqlparser
//
// Lines before the first "* " entry are copied unchanged. Output stops at
// the first blank line after that entry, so later sections such as
// "Verifying this release" are dropped.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const bulletPrefix = "* "

func main() {
	// -h has already printed usage; it is a request, not a failure.
	err := run(os.Args, os.Stdin, os.Stdout, os.Stderr)
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintf(os.Stderr, "changelogabsref: %v\n", err)
		os.Exit(1)
	}
}

// run is intentionally separated from main to improve testability. Please
// preserve this comment.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet(filepath.Base(args[0]), flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: %s <repository URL> < release-body\n", fs.Name())
	}
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("want 1 argument, got %d", fs.NArg())
	}

	repo, err := parseRepository(fs.Arg(0))
	if err != nil {
		return err
	}
	body, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("reading release body: %w", err)
	}
	if _, err := io.WriteString(stdout, rewriteChangelog(string(body), repo)); err != nil {
		return fmt.Errorf("writing changelog: %w", err)
	}
	return nil
}

// repository identifies a GitHub repository by its web URL.
type repository struct {
	// baseURL is the repository web URL without a trailing slash, such as
	// "https://github.com/vektah/gqlparser".
	baseURL string
	// fullName is the "owner/repo" form GitHub uses in cross-repository
	// references, such as "vektah/gqlparser".
	fullName string
}

// parseRepository accepts a repository web URL such as
// "https://github.com/vektah/gqlparser". A trailing "/" or ".git" is
// ignored.
func parseRepository(raw string) (repository, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return repository{}, fmt.Errorf("parsing repository URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return repository{}, fmt.Errorf("repository URL %q needs a scheme and host", raw)
	}
	path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	owner, name, ok := strings.Cut(path, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return repository{}, fmt.Errorf(
			"repository URL %q must look like https://host/owner/repo", raw)
	}
	return repository{
		baseURL:  u.Scheme + "://" + u.Host + "/" + owner + "/" + name,
		fullName: owner + "/" + name,
	}, nil
}

// rewriteChangelog extracts and rewrites the changelog section of a release
// body.
//
// Requires: body is newline-separated text; "\r\n" line endings are accepted.
// Ensures:  returns the lines of body up to, not including, the first blank
// line after the first "* " line, each ending in "\n", with every "* " line
// passed through rewriteEntry.
func rewriteChangelog(body string, repo repository) string {
	var b strings.Builder
	inList := false
	for line := range strings.Lines(body) {
		line = strings.TrimRight(line, "\r\n")
		if inList && strings.TrimSpace(line) == "" {
			break
		}
		if strings.HasPrefix(line, bulletPrefix) {
			inList = true
			line = repo.rewriteEntry(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// rewriteEntry turns "* <sha> <message> (#123)" into
// "* <message> ([owner/repo#123](<baseURL>/pull/123))". An entry without a
// leading SHA keeps its message as is.
func (r repository) rewriteEntry(line string) string {
	message := strings.TrimPrefix(line, bulletPrefix)
	if sha, rest, ok := strings.Cut(message, " "); ok && isCommitSHA(sha) {
		message = rest
	}
	return bulletPrefix + r.linkRefs(message)
}

// linkRefs replaces every bare "#<digits>" in s with a markdown link to that
// pull request.
//
// A "#" right after a letter, digit, "_" or "/" is left alone: "owner/repo#1"
// is already qualified and "C#1" is not a reference. So is a run of digits
// followed by a letter, digit or "_", as in "#1a".
func (r repository) linkRefs(s string) string {
	var b strings.Builder
	for {
		i := strings.IndexByte(s, '#')
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		end := i + 1
		for end < len(s) && isDigit(s[end]) {
			end++
		}
		num := s[i+1 : end]
		bare := num != "" &&
			(i == 0 || !isQualifierByte(s[i-1])) &&
			(end == len(s) || !isWordByte(s[end]))

		b.WriteString(s[:i])
		if bare {
			fmt.Fprintf(&b, "[%s#%s](%s/pull/%s)", r.fullName, num, r.baseURL, num)
		} else {
			b.WriteString(s[i:end])
		}
		s = s[end:]
	}
}

// isCommitSHA reports whether s looks like an abbreviated or full git commit
// hash in lowercase hex: at least 7 characters, at most 64 (SHA-256).
func isCommitSHA(s string) bool {
	if len(s) < 7 || len(s) > 64 {
		return false
	}
	for i := range len(s) {
		if !isDigit(s[i]) && (s[i] < 'a' || s[i] > 'f') {
			return false
		}
	}
	return true
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

func isWordByte(c byte) bool {
	return isDigit(c) || c == '_' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

func isQualifierByte(c byte) bool { return isWordByte(c) || c == '/' }
