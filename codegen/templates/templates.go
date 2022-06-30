package templates

import (
	"bytes"
	"fmt"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/99designs/gqlgen/internal/code"

	"github.com/99designs/gqlgen/internal/imports"
)

// CurrentImports keeps track of all the import declarations that are needed during the execution of a plugin.
// this is done with a global because subtemplates currently get called in functions. Lets aim to remove this eventually.
var CurrentImports *Imports

// Options specify various parameters to rendering a template.
type Options struct {
	// PackageName is a helper that specifies the package header declaration.
	// In other words, when you write the template you don't need to specify `package X`
	// at the top of the file. By providing PackageName in the Options, the Render
	// function will do that for you.
	PackageName string
	// Template is a string of the entire template that
	// will be parsed and rendered. If it's empty,
	// the plugin processor will look for .gotpl files
	// in the same directory of where you wrote the plugin.
	Template string

	// Use the go:embed API to collect all the template files you want to pass into Render
	// this is an alternative to passing the Template option
	TemplateFS fs.FS

	// Filename is the name of the file that will be
	// written to the system disk once the template is rendered.
	Filename        string
	RegionTags      bool
	GeneratedHeader bool
	// PackageDoc is documentation written above the package line
	PackageDoc string
	// FileNotice is notice written below the package line
	FileNotice string
	// Data will be passed to the template execution.
	Data  interface{}
	Funcs template.FuncMap

	// Packages cache, you can find me on config.Config
	Packages *code.Packages
}

// Render renders a gql plugin template from the given Options. Render is an
// abstraction of the text/template package that makes it easier to write gqlgen
// plugins. If Options.Template is empty, the Render function will look for `.gotpl`
// files inside the directory where you wrote the plugin.
func Render(cfg Options) error {
	if CurrentImports != nil {
		panic(fmt.Errorf("recursive or concurrent call to RenderToFile detected"))
	}
	CurrentImports = &Imports{packages: cfg.Packages, destDir: filepath.Dir(cfg.Filename)}

	funcs := Funcs()
	for n, f := range cfg.Funcs {
		funcs[n] = f
	}

	t := template.New("").Funcs(funcs)
	t, err := parseTemplates(cfg, t)
	if err != nil {
		return err
	}

	var allTemplates []string
	for _, template := range t.Templates() {
		// templates that end with _.gotpl are special files we don't want to include
		if strings.HasSuffix(template.Name(), "_.gotpl") ||
			// filter out templates added with {{ template xxx }} syntax inside the template file
			!strings.HasSuffix(template.Name(), ".gotpl") {
			continue
		}

		allTemplates = append(allTemplates, template.Name())
	}

	// then execute all the important looking ones in order, adding them to the same file
	sort.Slice(allTemplates, func(i, j int) bool {
		// important files go first
		if strings.HasSuffix(allTemplates[i], "!.gotpl") {
			return true
		}
		if strings.HasSuffix(allTemplates[j], "!.gotpl") {
			return false
		}
		return allTemplates[i] < allTemplates[j]
	})

	var buf bytes.Buffer
	for _, root := range allTemplates {
		if cfg.RegionTags {
			buf.WriteString("\n// region    " + center(70, "*", " "+root+" ") + "\n")
		}
		err := t.Lookup(root).Execute(&buf, cfg.Data)
		if err != nil {
			return fmt.Errorf("%s: %w", root, err)
		}
		if cfg.RegionTags {
			buf.WriteString("\n// endregion " + center(70, "*", " "+root+" ") + "\n")
		}
	}

	var result bytes.Buffer
	if cfg.GeneratedHeader {
		result.WriteString("// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.\n\n")
	}
	if cfg.PackageDoc != "" {
		result.WriteString(cfg.PackageDoc + "\n")
	}
	result.WriteString("package ")
	result.WriteString(cfg.PackageName)
	result.WriteString("\n\n")
	if cfg.FileNotice != "" {
		result.WriteString(cfg.FileNotice)
		result.WriteString("\n\n")
	}
	result.WriteString("import (\n")
	result.WriteString(CurrentImports.String())
	result.WriteString(")\n")
	_, err = buf.WriteTo(&result)
	if err != nil {
		return err
	}
	CurrentImports = nil

	err = write(cfg.Filename, result.Bytes(), cfg.Packages)
	if err != nil {
		return err
	}

	cfg.Packages.Evict(code.ImportPathForDir(filepath.Dir(cfg.Filename)))
	return nil
}

func parseTemplates(cfg Options, t *template.Template) (*template.Template, error) {
	if cfg.Template != "" {
		var err error
		t, err = t.New("template.gotpl").Parse(cfg.Template)
		if err != nil {
			return nil, fmt.Errorf("error with provided template: %w", err)
		}
		return t, nil
	}

	var fileSystem fs.FS
	if cfg.TemplateFS != nil {
		fileSystem = cfg.TemplateFS
	} else {
		// load path relative to calling source file
		_, callerFile, _, _ := runtime.Caller(1)
		rootDir := filepath.Dir(callerFile)
		fileSystem = os.DirFS(rootDir)
	}

	t, err := t.ParseFS(fileSystem, "*.gotpl")
	if err != nil {
		return nil, fmt.Errorf("locating templates: %w", err)
	}

	return t, nil
}

func center(width int, pad string, s string) string {
	if len(s)+2 > width {
		return s
	}
	lpad := (width - len(s)) / 2
	rpad := width - (lpad + len(s))
	return strings.Repeat(pad, lpad) + s + strings.Repeat(pad, rpad)
}

func Funcs() template.FuncMap {
	return template.FuncMap{
		"ucFirst":       UcFirst,
		"lcFirst":       LcFirst,
		"quote":         strconv.Quote,
		"rawQuote":      rawQuote,
		"dump":          Dump,
		"ref":           ref,
		"ts":            TypeIdentifier,
		"call":          Call,
		"prefixLines":   prefixLines,
		"notNil":        notNil,
		"reserveImport": CurrentImports.Reserve,
		"lookupImport":  CurrentImports.Lookup,
		"go":            ToGo,
		"goPrivate":     ToGoPrivate,
		"add": func(a, b int) int {
			return a + b
		},
		"render": func(filename string, tpldata interface{}) (*bytes.Buffer, error) {
			return render(resolveName(filename, 0), tpldata)
		},
	}
}

func UcFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func LcFirst(s string) string {
	if s == "" {
		return ""
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func isDelimiter(c rune) bool {
	return c == '-' || c == '_' || unicode.IsSpace(c)
}

func ref(p types.Type) string {
	return CurrentImports.LookupType(p)
}

var pkgReplacer = strings.NewReplacer(
	"/", "ᚋ",
	".", "ᚗ",
	"-", "ᚑ",
	"~", "א",
)

func TypeIdentifier(t types.Type) string {
	res := ""
	for {
		switch it := t.(type) {
		case *types.Pointer:
			t.Underlying()
			res += "ᚖ"
			t = it.Elem()
		case *types.Slice:
			res += "ᚕ"
			t = it.Elem()
		case *types.Named:
			res += pkgReplacer.Replace(it.Obj().Pkg().Path())
			res += "ᚐ"
			res += it.Obj().Name()
			return res
		case *types.Basic:
			res += it.Name()
			return res
		case *types.Map:
			res += "map"
			return res
		case *types.Interface:
			res += "interface"
			return res
		default:
			panic(fmt.Errorf("unexpected type %T", it))
		}
	}
}

func Call(p *types.Func) string {
	pkg := CurrentImports.Lookup(p.Pkg().Path())

	if pkg != "" {
		pkg += "."
	}

	if p.Type() != nil {
		// make sure the returned type is listed in our imports.
		ref(p.Type().(*types.Signature).Results().At(0).Type())
	}

	return pkg + p.Name()
}

func ToGo(name string) string {
	if name == "_" {
		return "_"
	}
	runes := make([]rune, 0, len(name))

	wordWalker(name, func(info *wordInfo) {
		word := info.Word
		if info.MatchCommonInitial {
			word = strings.ToUpper(word)
		} else if !info.HasCommonInitial {
			if strings.ToUpper(word) == word || strings.ToLower(word) == word {
				// FOO or foo → Foo
				// FOo → FOo
				word = UcFirst(strings.ToLower(word))
			}
		}
		runes = append(runes, []rune(word)...)
	})

	return string(runes)
}

func ToGoPrivate(name string) string {
	if name == "_" {
		return "_"
	}
	runes := make([]rune, 0, len(name))

	first := true
	wordWalker(name, func(info *wordInfo) {
		word := info.Word
		switch {
		case first:
			if strings.ToUpper(word) == word || strings.ToLower(word) == word {
				// ID → id, CAMEL → camel
				word = strings.ToLower(info.Word)
			} else {
				// ITicket → iTicket
				word = LcFirst(info.Word)
			}
			first = false
		case info.MatchCommonInitial:
			word = strings.ToUpper(word)
		case !info.HasCommonInitial:
			word = UcFirst(strings.ToLower(word))
		}
		runes = append(runes, []rune(word)...)
	})

	return sanitizeKeywords(string(runes))
}

type wordInfo struct {
	Word               string
	MatchCommonInitial bool
	HasCommonInitial   bool
}

// This function is based on the following code.
// https://github.com/golang/lint/blob/06c8688daad7faa9da5a0c2f163a3d14aac986ca/lint.go#L679
func wordWalker(str string, f func(*wordInfo)) {
	runes := []rune(strings.TrimFunc(str, isDelimiter))
	w, i := 0, 0 // index of start of word, scan
	hasCommonInitial := false
	for i+1 <= len(runes) {
		eow := false // whether we hit the end of a word
		switch {
		case i+1 == len(runes):
			eow = true
		case isDelimiter(runes[i+1]):
			// underscore; shift the remainder forward over any run of underscores
			eow = true
			n := 1
			for i+n+1 < len(runes) && isDelimiter(runes[i+n+1]) {
				n++
			}

			// Leave at most one underscore if the underscore is between two digits
			if i+n+1 < len(runes) && unicode.IsDigit(runes[i]) && unicode.IsDigit(runes[i+n+1]) {
				n--
			}

			copy(runes[i+1:], runes[i+n+1:])
			runes = runes[:len(runes)-n]
		case unicode.IsLower(runes[i]) && !unicode.IsLower(runes[i+1]):
			// lower->non-lower
			eow = true
		}
		i++

		// [w,i) is a word.
		word := string(runes[w:i])
		if !eow && commonInitialisms[word] && !unicode.IsLower(runes[i]) {
			// through
			// split IDFoo → ID, Foo
			// but URLs → URLs
		} else if !eow {
			if commonInitialisms[word] {
				hasCommonInitial = true
			}
			continue
		}

		matchCommonInitial := false
		if commonInitialisms[strings.ToUpper(word)] {
			hasCommonInitial = true
			matchCommonInitial = true
		}

		f(&wordInfo{
			Word:               word,
			MatchCommonInitial: matchCommonInitial,
			HasCommonInitial:   hasCommonInitial,
		})
		hasCommonInitial = false
		w = i
	}
}

var keywords = []string{
	"break",
	"default",
	"func",
	"interface",
	"select",
	"case",
	"defer",
	"go",
	"map",
	"struct",
	"chan",
	"else",
	"goto",
	"package",
	"switch",
	"const",
	"fallthrough",
	"if",
	"range",
	"type",
	"continue",
	"for",
	"import",
	"return",
	"var",
	"_",
}

// sanitizeKeywords prevents collisions with go keywords for arguments to resolver functions
func sanitizeKeywords(name string) string {
	for _, k := range keywords {
		if name == k {
			return name + "Arg"
		}
	}
	return name
}

// commonInitialisms is a set of common initialisms.
// Only add entries that are highly unlikely to be non-initialisms.
// For instance, "ID" is fine (Freudian code is rare), but "AND" is not.
var commonInitialisms = map[string]bool{
	"ACL":   true,
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"CSV":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ICMP":  true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"KVK":   true,
	"LHS":   true,
	"PDF":   true,
	"PGP":   true,
	"QPS":   true,
	"QR":    true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"SVG":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"UUID":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSRF":  true,
	"XSS":   true,
}

func rawQuote(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "`+\"`\"+`") + "`"
}

func notNil(field string, data interface{}) bool {
	v := reflect.ValueOf(data)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return false
	}
	val := v.FieldByName(field)

	return val.IsValid() && !val.IsNil()
}

func Dump(val interface{}) string {
	switch val := val.(type) {
	case int:
		return strconv.Itoa(val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%f", val)
	case string:
		return strconv.Quote(val)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return "nil"
	case []interface{}:
		var parts []string
		for _, part := range val {
			parts = append(parts, Dump(part))
		}
		return "[]interface{}{" + strings.Join(parts, ",") + "}"
	case map[string]interface{}:
		buf := bytes.Buffer{}
		buf.WriteString("map[string]interface{}{")
		var keys []string
		for key := range val {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			data := val[key]

			buf.WriteString(strconv.Quote(key))
			buf.WriteString(":")
			buf.WriteString(Dump(data))
			buf.WriteString(",")
		}
		buf.WriteString("}")
		return buf.String()
	default:
		panic(fmt.Errorf("unsupported type %T", val))
	}
}

func prefixLines(prefix, s string) string {
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}

func resolveName(name string, skip int) string {
	if name[0] == '.' {
		// load path relative to calling source file
		_, callerFile, _, _ := runtime.Caller(skip + 1)
		return filepath.Join(filepath.Dir(callerFile), name[1:])
	}

	// load path relative to this directory
	_, callerFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(callerFile), name)
}

func render(filename string, tpldata interface{}) (*bytes.Buffer, error) {
	t := template.New("").Funcs(Funcs())

	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	t, err = t.New(filepath.Base(filename)).Parse(string(b))
	if err != nil {
		panic(err)
	}

	buf := &bytes.Buffer{}
	return buf, t.Execute(buf, tpldata)
}

func write(filename string, b []byte, packages *code.Packages) error {
	err := os.MkdirAll(filepath.Dir(filename), 0o755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	formatted, err := imports.Prune(filename, b, packages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gofmt failed on %s: %s\n", filepath.Base(filename), err.Error())
		formatted = b
	}

	err = os.WriteFile(filename, formatted, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}
