// Gets the latest Apollo Embedded Sandbox Playground URL from the CDN S3 bucket
//
// To get the Subresource Integrity check, `go run main.go` and take what that outputs and run like
// this:
// CDN_FILE=https://embeddable-sandbox.cdn.apollographql.com/58165cf7452dbad480c7cb85e7acba085b3bac1d/embeddable-sandbox.umd.production.min.js
// curl -s $CDN_FILE | openssl dgst -sha256 -binary | openssl base64 -A; echo

package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"hash"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	apolloSandboxCdnUrl       = "https://embeddable-sandbox.cdn.apollographql.com"
	apolloSandboxSriAlgorithm = "sha256" // md5, sha256 or sha512
)

type ListBucketResult struct {
	XMLName               xml.Name `xml:"ListBucketResult"`
	Text                  string   `xml:",chardata"`
	Xmlns                 string   `xml:"xmlns,attr"`
	Name                  string   `xml:"Name"`
	Prefix                string   `xml:"Prefix"`
	NextContinuationToken string   `xml:"NextContinuationToken"`
	KeyCount              string   `xml:"KeyCount"`
	IsTruncated           bool     `xml:"IsTruncated"`
	Contents              []struct {
		Text           string    `xml:",chardata"`
		Key            string    `xml:"Key"`
		Generation     string    `xml:"Generation"`
		MetaGeneration string    `xml:"MetaGeneration"`
		LastModified   time.Time `xml:"LastModified"`
		ETag           string    `xml:"ETag"`
		Size           string    `xml:"Size"`
	} `xml:"Contents"`
}

func main() {
	if err := updateApolloSandbox(); err != nil {
		log.Fatalln(err.Error())
	}
}

// updateApolloSandbox finds the latest version of apollo sandbox js and updates the apollo_sandbox_playground.go.
func updateApolloSandbox() error {
	repoRootPath, err := findRepoRootPath()
	if err != nil {
		return fmt.Errorf("failed to find git directory: %w", err)
	}

	latestKey, err := findLastRelease()
	if err != nil {
		return fmt.Errorf("failed to parse base url: %w", err)
	}

	latestJsUrl, err := url.JoinPath(apolloSandboxCdnUrl, latestKey)
	if err != nil {
		return fmt.Errorf("failed to join url: %w", err)
	}

	latestJsSri, err := computeSRIHash(latestJsUrl, apolloSandboxSriAlgorithm)
	if err != nil {
		return fmt.Errorf("failed to compute latestJsSri hash: %w", err)
	}

	apolloSandBoxFile := filepath.Join(repoRootPath, "graphql", "playground", "apollo_sandbox_playground.go")

	goFileBytes, err := alterApolloSandboxContents(apolloSandBoxFile, latestJsUrl, latestJsSri)
	if err != nil {
		return fmt.Errorf("failed to alter apollo sandbox contents: %w", err)
	}

	if err := os.WriteFile(apolloSandBoxFile, goFileBytes, 0o644); err != nil {
		return fmt.Errorf("failed to write apollo sandbox contents: %w", err)
	}
	return nil
}

// findRepoRootPath returns the path that contains ".git" directory, based on the working directory.
// It starts at the working directory, and walks up the filesystem hierarchy until it finds a valid
// ".git" directory. If it can't retrieve the working directory, and can't find a ".git" directory
// it will return an error.
func findRepoRootPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	dir := wd
	for {
		if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil && fi.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("failed to find a .git directory starting from %s", wd)
		}

		dir = parent
	}
}

// findLastRelease Finds the latest release from the CDN bucket.
// Ignores the _latest, latest and v2 keys.
func findLastRelease() (string, error) {
	baseUrl, err := url.Parse(apolloSandboxCdnUrl)
	if err != nil {
		return "", fmt.Errorf("failed to parse base url: %w", err)
	}

	var continuationToken string
	var latestKey string
	var latestTime time.Time

	for {
		result, err := getBucketFiles(baseUrl, continuationToken)
		if err != nil {
			return "", fmt.Errorf("failed to get latest release: %w", err)
		}

		for _, content := range result.Contents {
			if strings.HasSuffix(content.Key, "/embeddable-sandbox.umd.production.min.js") &&
				!strings.HasPrefix(content.Key, "_latest/") &&
				!strings.HasPrefix(content.Key, "latest/") &&
				!strings.HasPrefix(content.Key, "v2/") {
				if latestTime.IsZero() || latestTime.Before(content.LastModified) {
					latestKey = content.Key
					latestTime = content.LastModified
				}
			}
		}

		if !result.IsTruncated {
			break
		}
		continuationToken = result.NextContinuationToken
	}

	return latestKey, nil
}

// getBucketFiles gets the file list from the CDN bucket.
func getBucketFiles(baseUrl *url.URL, continuationToken string) (ListBucketResult, error) {
	query := baseUrl.Query()
	query.Set("list-type", "2")
	if continuationToken != "" {
		query.Set("continuationToken", continuationToken)
	}
	baseUrl.RawQuery = query.Encode()

	resp, err := http.Get(baseUrl.String())
	if err != nil {
		return ListBucketResult{}, fmt.Errorf("client: could not make request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ListBucketResult{}, fmt.Errorf("client: could not read response body: %w", err)
	}

	var result ListBucketResult
	if err := xml.Unmarshal(data, &result); err != nil {
		return ListBucketResult{}, fmt.Errorf("failed to unmarshal xml response %w", err)
	}

	return result, nil
}

// computeSRIHash computes the SRI hash for the given URL.
// See https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity
func computeSRIHash(reqURL string, algo string) (string, error) {
	h, err := newHasher(algo)
	if err != nil {
		return "", err
	}

	resp, err := http.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("client: could not make request: %w", err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", fmt.Errorf("could not copy bytes into hash: %w", err)
	}

	return integrity(algo, h.Sum(nil)), nil
}

// newHasher creates a new hasher for the given algorithm.
func newHasher(algo string) (hash.Hash, error) {
	switch algo {
	case "md5":
		return md5.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported crypto algo: %q, use either md5, sha256 or sha512", algo)
	}
}

// integrity computes the SRI hash for the given bytes.
func integrity(algo string, sum []byte) string {
	encoded := base64.StdEncoding.EncodeToString(sum)
	return fmt.Sprintf("%s-%s", algo, encoded)
}

// alterApolloSandboxContents alters the apollo sandbox source code contents to use the latest JS URL and SRI.
func alterApolloSandboxContents(filename, latestJsUrl, latestJsSri string) ([]byte, error) {
	tokenFileSet := token.NewFileSet()
	node, err := parser.ParseFile(tokenFileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	var mainJsUpdated, mainSriUpdated bool
	for _, decl := range node.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			valSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valSpec.Names {
				switch name.Name {
				case "apolloSandboxMainJs":
					valSpec.Values[i] = &ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(latestJsUrl),
					}
					mainJsUpdated = true
				case "apolloSandboxMainSri":
					valSpec.Values[i] = &ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(latestJsSri),
					}
					mainSriUpdated = true
				}
			}
		}
	}
	if !mainJsUpdated || !mainSriUpdated {
		return nil, errors.New("failed to find apolloSandboxMainJs or apolloSandboxMainSri constants")
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, tokenFileSet, node); err != nil {
		return nil, fmt.Errorf("failed to format ast: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to format source: %w", err)
	}
	return formatted, nil
}
