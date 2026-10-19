package tablecodegen

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copies lists what this package copies from the codegen package, with the SHA-256 of each
// original as it was when it was copied. An original is a file of the module, relative to
// its root, or a declaration in a Go file, written file#Name or file#Type.Method, whose
// source from its keyword or name to its end, without its doc comment, is hashed.
var copies = []struct {
	// copy is the file of this package that holds the copy, and the copied declaration.
	copy string
	// original is what the copy was copied from.
	original string
	// sum is the SHA-256 of the original when it was copied.
	sum string
}{
	// The templates that table mode shares with the functions mode, with the templates of
	// table_.gotpl called where the modes differ and the methods of Data called for table
	// mode.
	{
		"api!.gotpl", "codegen/api!.gotpl",
		"48ada2ae3cf03cd5b224a485248e533b2ecaafea1f7c28e07d46c3cdb5296367",
	},
	{
		"args.gotpl", "codegen/args.gotpl",
		"3d0bc2ee595cc1b94449042ebba73ab3f59c20139efe46b693e0020364d000a5",
	},
	{
		"directives_.gotpl", "codegen/directives_.gotpl",
		"afed3f7c96cbc5db1fe7f574231d9a449410de4f2f03eb3d59149869e5816610",
	},
	{
		"field_.gotpl", "codegen/field.gotpl",
		"5d798a6fb94c8fbaa0b6a4ab04becac8d837fdaaabdf7e7668989ad394ccd837",
	},
	{
		"generated!.gotpl", "codegen/generated!.gotpl",
		"4291bc51bed248017f991ef5ba279f1dd812bbbcf264e3308423a34b6fcf4afe",
	},
	{
		"input_.gotpl", "codegen/input.gotpl",
		"d82f7c6337f55f5f3b11089ba4590e14052970f249f0585fa4ae36d1eaa64f3d",
	},
	{
		"interface.gotpl", "codegen/interface.gotpl",
		"7168a807f9e574815cff008ce53ca1a6508655b22d37f7f267aebf2a36667bb2",
	},
	{
		"internal!.gotpl", "codegen/internal!.gotpl",
		"f664fddc7495ff6ed85c13aac3f6d1db82d38ba643507bfc4c9c294a504f15b8",
	},
	{
		"type.gotpl", "codegen/type.gotpl",
		"c18dbf7d3c51be519eaf5db43280409c27abc367a68ca9c642b71120b02e477b",
	},
	// object_table.gotpl is not a copy, but resolves the fields as object.gotpl does.
	{
		"object_table.gotpl", "codegen/object.gotpl",
		"0d247faea1bc26bf856c6b89a7adb556f4c471a903b6cecd654194140534c120",
	},

	// The functions that write the files of the executor.
	{
		"incremental.go#GenerateCodeIncremental", "codegen/incremental.go#GenerateCodeIncremental",
		"4ba5484cb4d99a262a1457eb433c9bad73c16f5f3312e4548d50b306a6e65e20",
	},
	{
		"generate.go#generateCode", "codegen/generate.go#GenerateCode",
		"c0dea7fb532a94df393e9465eae91410b30980f758b292a2cfd5051408ef216c",
	},
	{
		"generate.go#generateSingleFile", "codegen/generate.go#generateSingleFile",
		"32841592c6ffc468dd0c47d769b97b37f23be0ea5e96b47acb88737298ca99a6",
	},
	{
		"generate.go#generatePerSchema", "codegen/generate.go#generatePerSchema",
		"3afe2a1a5f978c8a77cf49ba5c154738f9b5db7539699cbf74fe113024d43754",
	},
	{
		"generate.go#filename", "codegen/generate.go#filename",
		"8bb0c66f69159966a9259e523960a41f8678d7e22a26019756a98a3c2424d268",
	},
	{
		"generate.go#addBuild", "codegen/generate.go#addBuild",
		"da803c089889f0e8f9852ec9f194148b8c1b455e016a76df9b8d49a6471edaeb",
	},
	{
		"generate.go#generateRootFile", "codegen/generate.go#generateRootFile",
		"f7d47b40d20ffbc06cb74e3d377da9f2f854997e8bd3ab21cf7b69027832885f",
	},
	{
		"generate.go#addObjects", "codegen/generate.go#addObjects",
		"519fee32e881d4fad796ce771cda0c89c18ae882058e8ed91b4c8f5ba4ad5091",
	},
	{
		"generate.go#addInputs", "codegen/generate.go#addInputs",
		"19c4da3d69aa22f8c59537f7e657032bcdf940b0db0d5ff91f4fe88ac0a269a4",
	},
	{
		"generate.go#addInterfaces", "codegen/generate.go#addInterfaces",
		"7d8da96912e21b3403bd93b49e995a0ec3a6e7f2b6c55146acd4cdff93ac3c5b",
	},
	{
		"generate.go#addDirectives", "codegen/generate.go#addDirectives",
		"4fb595cb3334f93262a91a540969591362993988ce83cd6f398ab16f8fadcea7",
	},
	{
		"generate.go#addReferencedTypes", "codegen/generate.go#addReferencedTypes",
		"a8244f7df05d717ccddd70a0471ee3f71a69a6f60268c22ff91e58d476e971aa",
	},
	// copyData and addBuild copy the exported fields of codegen.Data.
	{
		"data.go#copyData", "codegen/data.go#Data",
		"394b9837a7db9ad0580a01ff6e0e3fa363f0183e42da8978d1fe6d0310cce3b8",
	},

	// The unexported functions of the codegen package that table mode calls.
	{
		"field.go#federationRequiresArgName", "codegen/field.go#federationRequiresArgName",
		"000089bfb055aae0db53fc1ecc047e21b3ac3e1a1892674f359a4ce75300487c",
	},
	{
		"field.go#callArgs", "codegen/field.go#Field.CallArgs",
		"114a0b100b31aa6a77780b48ccfbc470f325025debff22395d9344d846ddaf8d",
	},
	{
		"field.go#fieldArgExpression", "codegen/field.go#Field.fieldArgExpression",
		"434a1e91475ded651334fac4432c26e0850622dafd33fb4d395f7ce2f624b31f",
	},
	{
		"field.go#callArgExpressions", "codegen/field.go#Field.callArgExpressions",
		"c2f91f44f5ff46ec3ac2b17694950cbbdeb17687a8009266899ed12f8ef6d31d",
	},
	{
		"field.go#formatGoType", "codegen/field.go#formatGoType",
		"9253d45e8b4e5c48a523bf01b5fea9454ee445f3d0102bdd4786187435bb2752",
	},
	{
		"directive.go#locationDirectives", "codegen/directive.go#locationDirectives",
		"113a8b6917fd7798ab7a8994dafffbed782b6fac51fdba796c25b0b9e55830d2",
	},
	{
		"data.go#collectBatchFieldTypes", "codegen/data.go#collectBatchFieldTypes",
		"2a646d6b33ea2e35c4e793ad13b8f3b6b8b35c8b7167777b08f6f192a943ab9e",
	},
	{
		"data.go#batchImplementors", "codegen/data.go#Data.BatchImplementors",
		"7a452083ebf615f40479f008eb60f7f0d699732de651b33adcb980f791dead67",
	},
	// tableGoTypes finds what bindField binds the fields to.
	{
		"data.go#tableGoTypes", "codegen/field.go#builder.bindField",
		"a5547434569d03807813c8a6d4eecb333f275f23ac6fb8e019432784093c66c8",
	},
}

// TestCopies fails when an original of what this package copies from the codegen package
// changed since it was copied, so that the copy follows the change.
func TestCopies(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, c := range copies {
		// The copy is where copies says, so that a failure points at it.
		if _, err := originalSource(".", c.copy); err != nil {
			t.Errorf("the copy of %s, internal/tablecodegen/%s: %v", c.original, c.copy, err)
		}
		src, err := originalSource(root, c.original)
		if err != nil {
			t.Errorf("%s, copied to internal/tablecodegen/%s: %v", c.original, c.copy, err)
			continue
		}
		sum := sha256.Sum256(src)
		if got := hex.EncodeToString(sum[:]); got != c.sum {
			t.Errorf(
				"%s changed since it was copied to internal/tablecodegen/%s: make the same change to the copy, then set the sum of the copy in copies_test.go to %s",
				c.original,
				c.copy,
				got,
			)
		}
	}
}

// originalSource returns the source of original, a file or a declaration in a Go file of
// the module at root: see copies.
func originalSource(root, original string) ([]byte, error) {
	file, name, isDecl := strings.Cut(original, "#")
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return nil, err
	}
	// A checkout on Windows may end the lines with CRLF, which the sums do not.
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	if !isDecl {
		return b, nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, b, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	recv, fn, isMethod := strings.Cut(name, ".")
	if !isMethod {
		recv, fn = "", name
	}
	for _, decl := range f.Decls {
		var node ast.Node
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if decl.Name.Name == fn && receiverName(decl) == recv {
				node = decl
			}
		case *ast.GenDecl:
			if !isMethod {
				node = specNamed(decl, fn)
			}
		}
		if node != nil {
			return b[fset.Position(node.Pos()).Offset:fset.Position(node.End()).Offset], nil
		}
	}
	return nil, fmt.Errorf("no declaration of %s", name)
}

// receiverName returns the name of the type of the receiver of decl, or "" for a function.
func receiverName(decl *ast.FuncDecl) string {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return ""
	}
	typ := decl.Recv.List[0].Type
	if star, ok := typ.(*ast.StarExpr); ok {
		typ = star.X
	}
	if ident, ok := typ.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

// specNamed returns the spec of decl that declares name, or nil.
func specNamed(decl *ast.GenDecl, name string) ast.Node {
	for _, spec := range decl.Specs {
		switch spec := spec.(type) {
		case *ast.TypeSpec:
			if spec.Name.Name == name {
				return spec
			}
		case *ast.ValueSpec:
			for _, n := range spec.Names {
				if n.Name == name {
					return spec
				}
			}
		}
	}
	return nil
}
