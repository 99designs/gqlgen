package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/internal/code"
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
	require.Equal(t, "", ToGo(""))

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
	require.Equal(t, "", ToGoPrivate(""))
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
			input:    [][]string{{"MyEnumName", "Value"}, {"MyEnumName", "value"}, {"MyEnumName", "vALue"}, {"MyEnumName", "VALue"}},
			expected: []string{"MyEnumNameValue", "MyEnumNamevalue", "MyEnumNameVALue", "MyEnumNameVALue0"},
		},
		{
			input:    [][]string{{"MyEnumName", "TitleValue"}, {"MyEnumName", "title_value"}, {"MyEnumName", "title_Value"}, {"MyEnumName", "Title_Value"}},
			expected: []string{"MyEnumNameTitleValue", "MyEnumNametitle_value", "MyEnumNametitle_Value", "MyEnumNameTitle_Value"},
		},
		{
			input:    [][]string{{"MyEnumName", "TitleValue", "OtherValue"}},
			expected: []string{"MyEnumNameTitleValueOtherValue"},
		},
		{
			input:    [][]string{{"MyEnumName", "TitleValue", "OtherValue"}, {"MyEnumName", "title_value", "OtherValue"}},
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
			input:    [][]string{{"MyEnumName", "Value"}, {"MyEnumName", "value"}, {"MyEnumName", "vALue"}, {"MyEnumName", "VALue"}},
			expected: []string{"myEnumNameValue", "myEnumNamevalue", "myEnumNameVALue", "myEnumNameVALue0"},
		},
		{
			input:    [][]string{{"MyEnumName", "TitleValue"}, {"MyEnumName", "title_value"}, {"MyEnumName", "title_Value"}, {"MyEnumName", "Title_Value"}},
			expected: []string{"myEnumNameTitleValue", "myEnumNametitle_value", "myEnumNametitle_Value", "myEnumNameTitle_Value"},
		},
		{
			input:    [][]string{{"MyEnumName", "TitleValue", "OtherValue"}},
			expected: []string{"myEnumNameTitleValueOtherValue"},
		},
		{
			input:    [][]string{{"MyEnumName", "TitleValue", "OtherValue"}, {"MyEnumName", "title_value", "OtherValue"}},
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
			input:    makeInput("RelatedURLs"),
			expected: []*wordInfo{{Word: "Related"}, {WordOffset: 1, Word: "URLs", HasCommonInitial: true}},
		},
		{
			input:    makeInput("ImageIDs"),
			expected: []*wordInfo{{Word: "Image"}, {WordOffset: 1, Word: "IDs", HasCommonInitial: true}},
		},
		{
			input:    makeInput("FooID"),
			expected: []*wordInfo{{Word: "Foo"}, {WordOffset: 1, Word: "ID", HasCommonInitial: true, MatchCommonInitial: true}},
		},
		{
			input:    makeInput("IDFoo"),
			expected: []*wordInfo{{Word: "ID", HasCommonInitial: true, MatchCommonInitial: true}, {WordOffset: 1, Word: "Foo"}},
		},
		{
			input:    makeInput("FooASCII"),
			expected: []*wordInfo{{Word: "Foo"}, {WordOffset: 1, Word: "ASCII", HasCommonInitial: true, MatchCommonInitial: true}},
		},
		{
			input:    makeInput("ASCIIFoo"),
			expected: []*wordInfo{{Word: "ASCII", HasCommonInitial: true, MatchCommonInitial: true}, {WordOffset: 1, Word: "Foo"}},
		},
		{
			input:    makeInput("FooUTF8"),
			expected: []*wordInfo{{Word: "Foo"}, {WordOffset: 1, Word: "UTF8", HasCommonInitial: true, MatchCommonInitial: true}},
		},
		{
			input:    makeInput("UTF8Foo"),
			expected: []*wordInfo{{Word: "UTF8", HasCommonInitial: true, MatchCommonInitial: true}, {WordOffset: 1, Word: "Foo"}},
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
