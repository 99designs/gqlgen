package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToGo(t *testing.T) {
	require.Equal(t, "ToCamel", ToGo("TO_CAMEL"))
	require.Equal(t, "ToCamel", ToGo("to_camel"))
	require.Equal(t, "ToCamel", ToGo("toCamel"))
	require.Equal(t, "ToCamel", ToGo("ToCamel"))
	require.Equal(t, "ToCamel", ToGo("to-camel"))

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
	require.Equal(t, "", ToGo(""))

	require.Equal(t, "RelatedUrls", ToGo("RelatedUrls"))
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

	require.Equal(t, "rangeArg", ToGoPrivate("Range"))

	require.Equal(t, "a", ToGoPrivate("A"))
	require.Equal(t, "id", ToGoPrivate("ID"))
	require.Equal(t, "", ToGoPrivate(""))
}

func Test_wordWalker(t *testing.T) {

	type Result struct {
		Value            string
		HasCommonInitial bool
	}
	helper := func(str string) []*Result {
		resultList := []*Result{}
		wordWalker(str, func(word string, hasCommonInitial bool) {
			resultList = append(resultList, &Result{word, hasCommonInitial})
		})
		return resultList
	}

	require.Equal(t, []*Result{{Value: "TO"}, {Value: "CAMEL"}}, helper("TO_CAMEL"))
	require.Equal(t, []*Result{{Value: "to"}, {Value: "camel"}}, helper("to_camel"))
	require.Equal(t, []*Result{{Value: "to"}, {Value: "Camel"}}, helper("toCamel"))
	require.Equal(t, []*Result{{Value: "To"}, {Value: "Camel"}}, helper("ToCamel"))
	require.Equal(t, []*Result{{Value: "to"}, {Value: "camel"}}, helper("to-camel"))

	require.Equal(t, []*Result{{Value: "Related"}, {Value: "URLs", HasCommonInitial: true}}, helper("RelatedURLs"))
	require.Equal(t, []*Result{{Value: "Image"}, {Value: "IDs", HasCommonInitial: true}}, helper("ImageIDs"))
	require.Equal(t, []*Result{{Value: "Foo"}, {Value: "ID", HasCommonInitial: true}}, helper("FooID"))
	require.Equal(t, []*Result{{Value: "ID", HasCommonInitial: true}, {Value: "Foo"}}, helper("IDFoo"))
	require.Equal(t, []*Result{{Value: "Foo"}, {Value: "ASCII", HasCommonInitial: true}}, helper("FooASCII"))
	require.Equal(t, []*Result{{Value: "ASCII", HasCommonInitial: true}, {Value: "Foo"}}, helper("ASCIIFoo"))
	require.Equal(t, []*Result{{Value: "Foo"}, {Value: "UTF8", HasCommonInitial: true}}, helper("FooUTF8"))
	require.Equal(t, []*Result{{Value: "UTF8", HasCommonInitial: true}, {Value: "Foo"}}, helper("UTF8Foo"))

	require.Equal(t, []*Result{{Value: "A"}}, helper("A"))
	require.Equal(t, []*Result{{Value: "ID", HasCommonInitial: true}}, helper("ID"))
	require.Equal(t, []*Result{{Value: "ID", HasCommonInitial: true}}, helper("id"))
	require.Equal(t, []*Result{}, helper(""))

	require.Equal(t, []*Result{{Value: "Related"}, {Value: "Urls"}}, helper("RelatedUrls"))
}

func TestCenter(t *testing.T) {
	require.Equal(t, "fffff", center(3, "#", "fffff"))
	require.Equal(t, "##fffff###", center(10, "#", "fffff"))
	require.Equal(t, "###fffff###", center(11, "#", "fffff"))
}
