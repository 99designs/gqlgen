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
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
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
	var continuationToken string
	var latestKey string
	var latestTime time.Time
	isTruncated := true

	for isTruncated {
		continuationToken, isTruncated, latestKey, latestTime = getPage(
			latestKey,
			latestTime,
			continuationToken,
			isTruncated,
		)
	}

	cdnFileURL := fmt.Sprintf(
		"%s/%s",
		"https://embeddable-sandbox.cdn.apollographql.com",
		latestKey,
	)
	cdnFileBytes := getCDNFile(cdnFileURL)

	sri, err := fingerprintTransform(cdnFileBytes, "sha256")
	var gitRepoDir string
	gitRepoDir, err = findGitDirectory()
	if err != nil {
		fmt.Printf("Unable to findGitDirectory: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Please update:")
	apolloSandBoxFile := filepath.Join(
		filepath.Dir(gitRepoDir),
		"/graphql/playground/apollo_sandbox_playground.go",
	)

	goFileBytes, err := os.ReadFile(apolloSandBoxFile)
	if err != nil {
		fmt.Printf("Unable to ReadFile %s: %s\n", apolloSandBoxFile, err)
		os.Exit(1)
	}

	goFileBytes = alterApolloSandboxContents(goFileBytes, latestKey, sri)
	err = os.WriteFile(apolloSandBoxFile, goFileBytes, 0o644)
	if err != nil {
		log.Fatalln(err)
	}
	return
}

// TODO(steve): This isn't *quite* correct as it leaves the old stuff and SRI
// as well as adding it
func alterApolloSandboxContents(src []byte, latestKey, sri string) []byte {
	prefixLatestLine := `<script rel="preload" as="script" crossorigin="anonymous" integrity="{{.mainSRI}}" type="text/javascript" src="https://embeddable-sandbox.cdn.apollographql.com/`
	suffixSuffixLatestLine := `/embeddable-sandbox.umd.production.min.js"></script>`

	prefixSRILine := `"mainSRI":            "`
	suffixSRILine := `",`
	lines := strings.Split(string(src), "\n")

	var newlines []string
	for _, line := range lines {
		switch {
		case strings.Contains(line, prefixLatestLine):
			chunks := strings.SplitAfter(line, prefixLatestLine)
			chunks = slices.Insert(chunks, 1, latestKey)
			chunks = append(chunks, suffixSuffixLatestLine)
			newlines = append(newlines, strings.Join(chunks, ""))
		case strings.Contains(line, prefixSRILine):
			chunks := strings.SplitAfter(line, prefixSRILine)
			chunks = slices.Insert(chunks, 1, sri)
			chunks = append(chunks, suffixSRILine)
			newlines = append(newlines, strings.Join(chunks, ""))
		default:
			newlines = append(newlines, line)
		}
	}
	output := strings.Join(newlines, "\n")
	return []byte(output)
}

func getCDNFile(reqURL string) []byte {
	resp, err := http.Get(reqURL)
	if err != nil {
		fmt.Printf("client: could not make request: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	return data
}

func getPage(
	latestKey string,
	latestTime time.Time,
	continuationToken string,
	isTruncated bool,
) (string, bool, string, time.Time) {
	const baseURL = "https://embeddable-sandbox.cdn.apollographql.com/?list-type=2"
	reqURL := baseURL
	if continuationToken != "" {
		reqURL += "&continuation-token=" + continuationToken
	}
	var result ListBucketResult
	resp, err := http.Get(reqURL)
	if err != nil {
		fmt.Printf("client: could not make request: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}

	err = xml.Unmarshal(data, &result)
	if err != nil {
		log.Fatalf("xml.Unmarshal failed with '%s'\n", err)
	}
	continuationToken = result.NextContinuationToken
	isTruncated = result.IsTruncated
	for _, content := range result.Contents {
		if strings.Contains(content.Key, "embeddable-sandbox.umd.production.min.js") &&
			!strings.Contains(content.Key, "embeddable-sandbox.umd.production.min.js.map") &&
			!strings.Contains(content.Key, "_latest") {
			if latestTime.IsZero() || latestTime.Before(content.LastModified) {
				latestKey = content.Key
				latestTime = content.LastModified
			}
		}
	}
	return continuationToken, isTruncated, latestKey, latestTime
}

const defaultHashAlgo = "sha256"

// Fingerprint applies fingerprinting of the given resource and hash algorithm.
// It defaults to sha256 if none given, and the options are md5, sha256 or sha512.
// The same algo is used for both the fingerprinting part (aka cache busting) and
// the base64-encoded Subresource Integrity hash, so you will have to stay away from
// md5 if you plan to use both.
// See https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity
// Transform creates a MD5 hash of the Resource content and inserts that hash before
// the extension in the filename.
func fingerprintTransform(src []byte, algo string) (string, error) {
	var h hash.Hash

	switch algo {
	case "md5":
		h = md5.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return "",
			fmt.Errorf("unsupported crypto algo: %q, use either md5, sha256 or sha512", algo)
	}

	buf := bytes.NewBuffer(src)
	_, err := io.Copy(h, buf)
	if err != nil {
		fmt.Printf("could not copy bytes into hash: %s\n", err)
		os.Exit(1)
	}
	var d []byte
	d, err = digest(h)
	if err != nil {
		return "", err
	}

	sri := integrity(algo, d)
	// digestString := hex.EncodeToString(d[:])
	return sri, nil
}

func integrity(algo string, sum []byte) string {
	encoded := base64.StdEncoding.EncodeToString(sum)
	return fmt.Sprintf("%s-%s", algo, encoded)
}

func digest(h hash.Hash) ([]byte, error) {
	sum := h.Sum(nil)
	return sum, nil
}

// findGitDirectory returns the path of the local ".git" directory, based on the working directory.
// It starts at the working directory, and walks up the filesystem hierarchy until it finds a valid
// ".git" directory. If it can't retrieve the working directory, and can't find a ".git" directory
// it will return an error.
func findGitDirectory() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	dir := wd
	for {
		fi, _ := os.Stat(filepath.Join(dir, ".git", "config"))
		if fi != nil && !fi.IsDir() {
			return filepath.Join(dir, ".git"), nil
		}

		if len(dir) == 0 || (len(dir) == 1 && os.IsPathSeparator(dir[0])) {
			return "", fmt.Errorf("failed to find a .git directory starting from %s", wd)
		}

		dir = strings.TrimSuffix(dir, string(os.PathSeparator))
		dir = filepath.Dir(dir)
	}
}
