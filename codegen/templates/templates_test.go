package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/internal/code"
	"github.com/99designs/gqlgen/internal/imports"
)

//go:embed *.gotpl
var templateFS embed.FS

func TestToGo(t *testing.T) {
	require.Equal(t, "ToCamel", ToGo("TO_CAMEL"))
	require.Equal(t, "ToCamel", ToGo("to_camel"))
	require.Equal(t, "ToCamel", ToGo("toCamel"))
	require.Equal(t, "ToCamel", ToGo("ToCamel"))
	require.Equal(t, "ToCamel", ToGo("to-camel"))
	require.Equal(t, "ToCamel", ToGo("-to-camel"))
	require.Equal(t, "ToCamel", ToGo("_to-camel"))
	require.Equal(t, "_", ToGo("_"))

	require.Equal(t, "RelatedURLs", ToGo("RelatedURLs"))
	require.Equal(t, "ImageIDs", ToGo("ImageIDs"))
	require.Equal(t, "FooID", ToGo("FooID"))
	require.Equal(t, "IDFoo", ToGo("IDFoo"))
	require.Equal(t, "FooASCII", ToGo("FooASCII"))
	require.Equal(t, "ASCIIFoo", ToGo("ASCIIFoo"))
	require.Equal(t, "FooUTF8", ToGo("FooUTF8"))
	require.Equal(t, "UTF8Foo", ToGo("UTF8Foo"))
	require.Equal(t, "JSONEncoding", ToGo("JSONEncoding"))

	require.Equal(t, "A", ToGo("A"))
	require.Equal(t, "ID", ToGo("ID"))
	require.Equal(t, "ID", ToGo("id"))
	require.Empty(t, ToGo(""))

	require.Equal(t, "RelatedUrls", ToGo("RelatedUrls"))
	require.Equal(t, "ITicket", ToGo("ITicket"))
	require.Equal(t, "FooTicket", ToGo("fooTicket"))

	require.Equal(t, "Idle", ToGo("IDLE"))
	require.Equal(t, "Idle", ToGo("Idle"))
	require.Equal(t, "Idle", ToGo("idle"))
	require.Equal(t, "Identities", ToGo("IDENTITIES"))
	require.Equal(t, "Identities", ToGo("Identities"))
	require.Equal(t, "Identities", ToGo("identities"))
	require.Equal(t, "Iphone", ToGo("IPHONE"))
	require.Equal(t, "IPhone", ToGo("iPHONE"))
	require.Equal(t, "UserIdentity", ToGo("USER_IDENTITY"))
	require.Equal(t, "UserIdentity", ToGo("UserIdentity"))
	require.Equal(t, "UserIdentity", ToGo("userIdentity"))
}

func TestToGoPrivate(t *testing.T) {
	require.Equal(t, "toCamel", ToGoPrivate("TO_CAMEL"))
	require.Equal(t, "toCamel", ToGoPrivate("to_camel"))
	require.Equal(t, "toCamel", ToGoPrivate("toCamel"))
	require.Equal(t, "toCamel", ToGoPrivate("ToCamel"))
	require.Equal(t, "toCamel", ToGoPrivate("to-camel"))

	require.Equal(t, "relatedURLs", ToGoPrivate("RelatedURLs"))
	require.Equal(t, "imageIDs", ToGoPrivate("ImageIDs"))
	require.Equal(t, "fooID", ToGoPrivate("FooID"))
	require.Equal(t, "idFoo", ToGoPrivate("IDFoo"))
	require.Equal(t, "fooASCII", ToGoPrivate("FooASCII"))
	require.Equal(t, "asciiFoo", ToGoPrivate("ASCIIFoo"))
	require.Equal(t, "fooUTF8", ToGoPrivate("FooUTF8"))
	require.Equal(t, "utf8Foo", ToGoPrivate("UTF8Foo"))
	require.Equal(t, "jsonEncoding", ToGoPrivate("JSONEncoding"))

	require.Equal(t, "relatedUrls", ToGoPrivate("RelatedUrls"))
	require.Equal(t, "iTicket", ToGoPrivate("ITicket"))

	require.Equal(t, "rangeArg", ToGoPrivate("Range"))

	require.Equal(t, "a", ToGoPrivate("A"))
	require.Equal(t, "id", ToGoPrivate("ID"))
	require.Equal(t, "id", ToGoPrivate("id"))
	require.Empty(t, ToGoPrivate(""))
	require.Equal(t, "_", ToGoPrivate("_"))

	require.Equal(t, "idle", ToGoPrivate("IDLE"))
	require.Equal(t, "idle", ToGoPrivate("Idle"))
	require.Equal(t, "idle", ToGoPrivate("idle"))
	require.Equal(t, "identities", ToGoPrivate("IDENTITIES"))
	require.Equal(t, "identities", ToGoPrivate("Identities"))
	require.Equal(t, "identities", ToGoPrivate("identities"))
	require.Equal(t, "iphone", ToGoPrivate("IPHONE"))
	require.Equal(t, "iPhone", ToGoPrivate("iPHONE"))
}

func TestToGoModelName(t *testing.T) {
	type aTest struct {
		input    [][]string
		expected []string
	}

	theTests := []aTest{
		{
			input:    [][]string{{"MyValue"}},
			expected: []string{"MyValue"},
		},
		{
			input:    [][]string{{"MyValue"}, {"myValue"}},
			expected: []string{"MyValue", "MyValue0"},
		},
		{
			input:    [][]string{{"MyValue"}, {"YourValue"}},
			expected: []string{"MyValue", "YourValue"},
		},
		{
			input:    [][]string{{"MyEnumName", "Value"}},
			expected: []string{"MyEnumNameValue"},
		},
		{
			input:    [][]string{{"MyEnumName", "Value"}, {"MyEnumName", "value"}},
			expected: []string{"MyEnumNameValue", "MyEnumNamevalue"},
		},
		{
			input:    [][]string{{"MyEnumName", "value"}, {"MyEnumName", "Value"}},
			expected: []string{"MyEnumNameValue", "MyEnumNameValue0"},
		},
		{
			input: [][]string{
				{"MyEnumName", "Value"},
				{"MyEnumName", "value"},
				{"MyEnumName", "vALue"},
				{"MyEnumName", "VALue"},
			},
			expected: []string{
				"MyEnumNameValue",
				"MyEnumNamevalue",
				"MyEnumNameVALue",
				"MyEnumNameVALue0",
			},
		},
		{
			input: [][]string{
				{"MyEnumName", "TitleValue"},
				{"MyEnumName", "title_value"},
				{"MyEnumName", "title_Value"},
				{"MyEnumName", "Title_Value"},
			},
			expected: []string{
				"MyEnumNameTitleValue",
				"MyEnumNametitle_value",
				"MyEnumNametitle_Value",
				"MyEnumNameTitle_Value",
			},
		},
		{
			input:    [][]string{{"MyEnumName", "TitleValue", "OtherValue"}},
			expected: []string{"MyEnumNameTitleValueOtherValue"},
		},
		{
			input: [][]string{
				{"MyEnumName", "TitleValue", "OtherValue"},
				{"MyEnumName", "title_value", "OtherValue"},
			},
			expected: []string{"MyEnumNameTitleValueOtherValue", "MyEnumNametitle_valueOtherValue"},
		},
	}

	for ti, at := range theTests {
		resetModelNames()
		t.Run(fmt.Sprintf("modelname-%d", ti), func(t *testing.T) {
			at := at
			for i, n := range at.input {
				require.Equal(t, at.expected[i], ToGoModelName(n...))
			}
		})
	}
}

func TestToGoPrivateModelName(t *testing.T) {
	type aTest struct {
		input    [][]string
		expected []string
	}

	theTests := []aTest{
		{
			input:    [][]string{{"MyValue"}},
			expected: []string{"myValue"},
		},
		{
			input:    [][]string{{"MyValue"}, {"myValue"}},
			expected: []string{"myValue", "myValue0"},
		},
		{
			input:    [][]string{{"MyValue"}, {"YourValue"}},
			expected: []string{"myValue", "yourValue"},
		},
		{
			input:    [][]string{{"MyEnumName", "Value"}},
			expected: []string{"myEnumNameValue"},
		},
		{
			input:    [][]string{{"MyEnumName", "Value"}, {"MyEnumName", "value"}},
			expected: []string{"myEnumNameValue", "myEnumNamevalue"},
		},
		{
			input:    [][]string{{"MyEnumName", "value"}, {"MyEnumName", "Value"}},
			expected: []string{"myEnumNameValue", "myEnumNameValue0"},
		},
		{
			input: [][]string{
				{"MyEnumName", "Value"},
				{"MyEnumName", "value"},
				{"MyEnumName", "vALue"},
				{"MyEnumName", "VALue"},
			},
			expected: []string{
				"myEnumNameValue",
				"myEnumNamevalue",
				"myEnumNameVALue",
				"myEnumNameVALue0",
			},
		},
		{
			input: [][]string{
				{"MyEnumName", "TitleValue"},
				{"MyEnumName", "title_value"},
				{"MyEnumName", "title_Value"},
				{"MyEnumName", "Title_Value"},
			},
			expected: []string{
				"myEnumNameTitleValue",
				"myEnumNametitle_value",
				"myEnumNametitle_Value",
				"myEnumNameTitle_Value",
			},
		},
		{
			input:    [][]string{{"MyEnumName", "TitleValue", "OtherValue"}},
			expected: []string{"myEnumNameTitleValueOtherValue"},
		},
		{
			input: [][]string{
				{"MyEnumName", "TitleValue", "OtherValue"},
				{"MyEnumName", "title_value", "OtherValue"},
			},
			expected: []string{"myEnumNameTitleValueOtherValue", "myEnumNametitle_valueOtherValue"},
		},
	}

	for ti, at := range theTests {
		resetModelNames()
		t.Run(fmt.Sprintf("modelname-%d", ti), func(t *testing.T) {
			at := at
			for i, n := range at.input {
				require.Equal(t, at.expected[i], ToGoPrivateModelName(n...))
			}
		})
	}
}

func Test_wordWalker(t *testing.T) {
	makeInput := func(str string) []*wordInfo {
		resultList := make([]*wordInfo, 0)
		wordWalker(str, func(info *wordInfo) {
			resultList = append(resultList, info)
		})
		return resultList
	}

	type aTest struct {
		expected []*wordInfo
		input    []*wordInfo
	}

	theTests := []aTest{
		{
			input:    makeInput("TO_CAMEL"),
			expected: []*wordInfo{{Word: "TO"}, {WordOffset: 1, Word: "CAMEL"}},
		},
		{
			input:    makeInput("to_camel"),
			expected: []*wordInfo{{Word: "to"}, {WordOffset: 1, Word: "camel"}},
		},
		{
			input:    makeInput("toCamel"),
			expected: []*wordInfo{{Word: "to"}, {WordOffset: 1, Word: "Camel"}},
		},
		{
			input:    makeInput("ToCamel"),
			expected: []*wordInfo{{Word: "To"}, {WordOffset: 1, Word: "Camel"}},
		},
		{
			input:    makeInput("to-camel"),
			expected: []*wordInfo{{Word: "to"}, {WordOffset: 1, Word: "camel"}},
		},
		{
			input: makeInput("RelatedURLs"),
			expected: []*wordInfo{
				{Word: "Related"},
				{WordOffset: 1, Word: "URLs", HasCommonInitial: true},
			},
		},
		{
			input: makeInput("ImageIDs"),
			expected: []*wordInfo{
				{Word: "Image"},
				{WordOffset: 1, Word: "IDs", HasCommonInitial: true},
			},
		},
		{
			input: makeInput("FooID"),
			expected: []*wordInfo{
				{Word: "Foo"},
				{WordOffset: 1, Word: "ID", HasCommonInitial: true, MatchCommonInitial: true},
			},
		},
		{
			input: makeInput("IDFoo"),
			expected: []*wordInfo{
				{Word: "ID", HasCommonInitial: true, MatchCommonInitial: true},
				{WordOffset: 1, Word: "Foo"},
			},
		},
		{
			input: makeInput("FooASCII"),
			expected: []*wordInfo{
				{Word: "Foo"},
				{WordOffset: 1, Word: "ASCII", HasCommonInitial: true, MatchCommonInitial: true},
			},
		},
		{
			input: makeInput("ASCIIFoo"),
			expected: []*wordInfo{
				{Word: "ASCII", HasCommonInitial: true, MatchCommonInitial: true},
				{WordOffset: 1, Word: "Foo"},
			},
		},
		{
			input: makeInput("FooUTF8"),
			expected: []*wordInfo{
				{Word: "Foo"},
				{WordOffset: 1, Word: "UTF8", HasCommonInitial: true, MatchCommonInitial: true},
			},
		},
		{
			input: makeInput("UTF8Foo"),
			expected: []*wordInfo{
				{Word: "UTF8", HasCommonInitial: true, MatchCommonInitial: true},
				{WordOffset: 1, Word: "Foo"},
			},
		},
		{
			input:    makeInput("A"),
			expected: []*wordInfo{{Word: "A"}},
		},
		{
			input:    makeInput("ID"),
			expected: []*wordInfo{{Word: "ID", HasCommonInitial: true, MatchCommonInitial: true}},
		},
		{
			input:    makeInput("id"),
			expected: []*wordInfo{{Word: "id", HasCommonInitial: true, MatchCommonInitial: true}},
		},
		{
			input:    makeInput(""),
			expected: make([]*wordInfo, 0),
		},
		{
			input:    makeInput("RelatedUrls"),
			expected: []*wordInfo{{Word: "Related"}, {WordOffset: 1, Word: "Urls"}},
		},
		{
			input:    makeInput("USER_IDENTITY"),
			expected: []*wordInfo{{Word: "USER"}, {WordOffset: 1, Word: "IDENTITY"}},
		},
		{
			input:    makeInput("ITicket"),
			expected: []*wordInfo{{Word: "ITicket"}},
		},
	}

	for i, at := range theTests {
		t.Run(fmt.Sprintf("wordWalker-%d", i), func(t *testing.T) {
			require.Equal(t, at.expected, at.input)
		})
	}
}

func TestCenter(t *testing.T) {
	require.Equal(t, "fffff", center(3, "#", "fffff"))
	require.Equal(t, "##fffff###", center(10, "#", "fffff"))
	require.Equal(t, "###fffff###", center(11, "#", "fffff"))
}

func TestTemplateOverride(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "gqlgen")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	err = Render(Options{Template: "hello", Filename: f.Name(), Packages: code.NewPackages()})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRenderFS(t *testing.T) {
	tempDir := t.TempDir()

	outDir := filepath.Join(tempDir, "output")

	_ = os.Mkdir(outDir, 0o755)

	f, err := os.CreateTemp(outDir, "gqlgen.go")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	err = Render(Options{TemplateFS: templateFS, Filename: f.Name(), Packages: code.NewPackages()})
	if err != nil {
		t.Fatal(err)
	}

	expectedString := "package \n\nimport (\n)\nthis is my test package"
	actualContents, _ := os.ReadFile(f.Name())
	actualContentsStr := string(actualContents)

	// don't look at last character since it's \n on Linux and \r\n on Windows
	assert.Equal(t, expectedString, actualContentsStr[:len(expectedString)])
}

func TestDict(t *testing.T) {
	tests := []struct {
		name      string
		input     []any
		expected  map[string]any
		expectErr bool
	}{
		{
			name:      "valid key-value pairs",
			input:     []any{"key1", "value1", "key2", "value2"},
			expected:  map[string]any{"key1": "value1", "key2": "value2"},
			expectErr: false,
		},
		{
			name:      "odd number of arguments",
			input:     []any{"key1", "value1", "key2"},
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "non-string key",
			input:     []any{"key1", "value1", 123, "value2"},
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "empty input",
			input:     []any{},
			expected:  map[string]any{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dict(tt.input...)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// writeContent is a small helper that renders a fixed, gofmt-clean Go file
// through write() so the atomic-write tests exercise the real code path
// (formatting, unchanged-content short-circuit, temp-then-rename).
func writeContent(t *testing.T, filename string, content string) {
	t.Helper()
	packages := code.NewPackages()
	require.NoError(t, write(filename, []byte(content), packages, imports.PruneOptions{}))
}

// TestWriteIsAtomicPreservesPermissionsAndLeavesNoTemp covers the three
// properties the atomic write must hold, mirroring the reference
// implementations (google/renameio, natefinch/atomic, tailscale/atomicfile,
// moby/sys) that this change was compared against in #4262:
//
//  1. Existing destination mode is preserved across a regen (not silently
//     dropped to the 0o600 that os.CreateTemp gives the temp file).
//  2. A brand-new file is created with 0o644 — the same mode os.WriteFile
//     used here before this change — not 0o600.
//  3. No temp file is left behind in the directory after a successful write.
func TestWriteIsAtomicPreservesPermissionsAndLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()

	goFile := "package graph\n\n// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.\n\nfunc a() {}\n"

	// Case 1: pre-existing file with a non-default mode must keep that mode.
	existing := filepath.Join(dir, "existing.go")
	require.NoError(t, os.WriteFile(existing, []byte("package graph\n\nfunc a() {}\n"), 0o600))
	before, err := os.Stat(existing)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), before.Mode().Perm())

	writeContent(t, existing, goFile)

	after, err := os.Stat(existing)
	require.NoError(t, err)
	require.Equal(t, before.Mode().Perm(), after.Mode().Perm(),
		"existing file mode must be preserved across a regen, not dropped to 0o600 or 0o644")

	// Case 2: brand-new file defaults to 0o644 (matches the old os.WriteFile mode).
	fresh := filepath.Join(dir, "fresh.go")
	writeContent(t, fresh, goFile)
	freshInfo, err := os.Stat(fresh)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), freshInfo.Mode().Perm(),
		"new file must be 0o644 (the pre-change os.WriteFile mode), not 0o600")

	// Case 3: no temp file left behind.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		require.False(t, strings.HasSuffix(e.Name(), ".tmp"),
			"leftover temp file after successful write: %q", e.Name())
	}
}

// TestWriteIsAtomicAndUnchangedShortCircuit confirms the unchanged-content
// short-circuit (which preserves mtime for the Go build cache) still fires and
// still writes atomically when content actually changes.
func TestWriteIsAtomicAndUnchangedShortCircuit(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "gen.go")
	content := "package graph\n\nfunc a() {}\n"

	writeContent(t, target, content)

	first, err := os.Stat(target)
	require.NoError(t, err)

	// Identical content → short-circuit: mtime should not move forward.
	writeContent(t, target, content)
	second, err := os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, first.ModTime(), second.ModTime(),
		"unchanged-content short-circuit must preserve mtime for the Go build cache")

	// Different content → rewritten atomically, content updates, mode unchanged.
	writeContent(t, target, "package graph\n\nfunc b() {}\n")
	got, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Contains(t, string(got), "func b()")
	mode, err := os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), mode.Mode().Perm())
}
