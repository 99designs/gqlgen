package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

const gqlparserURL = "https://github.com/vektah/gqlparser"

func TestRun(t *testing.T) {
	cases := map[string]struct {
		args  []string
		stdin string
		want  string
	}{
		"goreleaser release body": {
			args: []string{gqlparserURL},
			stdin: "## Changelog\n" +
				"* 0ffc5c674208302cbefd3f3cf46f98c1589ff0ab build(deps): bump the actions-deps group (#497)\n" +
				"* 993d81b3d06c35a2394f6ccc65589fe09e8ea8e1 validator: accept integer JSON numbers for ID variables (#496)\n" +
				"\n" +
				"## Verifying this release\n" +
				"\n" +
				"* not part of the changelog (#1)\n",
			want: "## Changelog\n" +
				"* build(deps): bump the actions-deps group ([vektah/gqlparser#497](https://github.com/vektah/gqlparser/pull/497))\n" +
				"* validator: accept integer JSON numbers for ID variables ([vektah/gqlparser#496](https://github.com/vektah/gqlparser/pull/496))\n",
		},
		"crlf line endings and whitespace-only terminator": {
			args:  []string{gqlparserURL},
			stdin: "* abc1234 fix (#1)\r\n \t\r\nrest\r\n",
			want:  "* fix ([vektah/gqlparser#1](https://github.com/vektah/gqlparser/pull/1))\n",
		},
		"blank lines before the first entry are kept": {
			args:  []string{gqlparserURL},
			stdin: "\n## Changelog\n\n* fix (#1)\n",
			want:  "\n## Changelog\n\n* fix ([vektah/gqlparser#1](https://github.com/vektah/gqlparser/pull/1))\n",
		},
		"entry without a sha": {
			args:  []string{gqlparserURL},
			stdin: "* Update README (#7)",
			want:  "* Update README ([vektah/gqlparser#7](https://github.com/vektah/gqlparser/pull/7))\n",
		},
		"non-hex first word is not a sha": {
			args:  []string{gqlparserURL},
			stdin: "* deadbeefz is a word\n",
			want:  "* deadbeefz is a word\n",
		},
		"several references in one entry": {
			args:  []string{gqlparserURL},
			stdin: "* abc1234 fix (#1, #2)\n",
			want: "* fix ([vektah/gqlparser#1](https://github.com/vektah/gqlparser/pull/1), " +
				"[vektah/gqlparser#2](https://github.com/vektah/gqlparser/pull/2))\n",
		},
		"qualified and non-reference hashes are left alone": {
			args:  []string{gqlparserURL},
			stdin: "* abc1234 see foo/bar#3, C#4, #5a, # and ##\n",
			want:  "* see foo/bar#3, C#4, #5a, # and ##\n",
		},
		"trailing slash and .git suffix on the URL": {
			args:  []string{gqlparserURL + ".git/"},
			stdin: "* fix (#1)\n",
			want:  "* fix ([vektah/gqlparser#1](https://github.com/vektah/gqlparser/pull/1))\n",
		},
		"empty input": {
			args:  []string{gqlparserURL},
			stdin: "",
			want:  "",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			args := append([]string{"changelogabsref"}, tc.args...)
			err := run(args, strings.NewReader(tc.stdin), &stdout, io.Discard)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if got := stdout.String(); got != tc.want {
				t.Errorf("stdout:\n got %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestRunErrors(t *testing.T) {
	cases := map[string]struct {
		args    []string
		wantErr string
	}{
		"no arguments": {args: nil, wantErr: "want 1 argument, got 0"},
		"two arguments": {
			args:    []string{gqlparserURL, "extra"},
			wantErr: "want 1 argument, got 2",
		},
		"missing scheme": {
			args:    []string{"github.com/vektah/gqlparser"},
			wantErr: "needs a scheme and host",
		},
		"missing repo": {
			args:    []string{"https://github.com/vektah"},
			wantErr: "must look like",
		},
		"too many segments": {
			args:    []string{"https://github.com/a/b/c"},
			wantErr: "must look like",
		},
		"unknown flag": {
			args:    []string{"-x", gqlparserURL},
			wantErr: "flag provided but not defined",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			args := append([]string{"changelogabsref"}, tc.args...)
			err := run(args, strings.NewReader("* fix (#1)\n"), &stdout, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("run error = %v, want it to contain %q", err, tc.wantErr)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	cases := map[string]struct {
		args []string
	}{
		"short": {args: []string{"-h"}},
		"long":  {args: []string{"--help"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var stderr bytes.Buffer
			args := append([]string{"changelogabsref"}, tc.args...)
			err := run(args, strings.NewReader(""), io.Discard, &stderr)
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("run error = %v, want flag.ErrHelp", err)
			}
			if !strings.Contains(stderr.String(), "usage: changelogabsref") {
				t.Errorf("stderr = %q, want usage", stderr.String())
			}
		})
	}
}
