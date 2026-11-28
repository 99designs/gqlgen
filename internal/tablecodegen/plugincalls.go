package tablecodegen

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	gqlast "github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
)

// Plugins that generate code into the package of the executor, such as the federation
// plugin, call the marshalers and unmarshalers of the functions mode by their names, as
// methods of the execution context or as functions that take it, as the config says.
// Table mode keeps those marshalers and unmarshalers typed and generates functions with
// those names that call them, so that the code of the plugins compiles in either mode.

// callForms is how code calls a marshaler or an unmarshaler of the functions mode.
type callForms struct {
	// method is set when the code calls it as a method of the execution context.
	method bool
	// function is set when the code calls it as a function that takes the execution
	// context.
	function bool
}

// pluginCallable reports whether code may call the marshaler of t by the name that the
// functions mode gives it, with the signature that table mode generates for it: the
// marshaler of the subscription type returns the context of the event as well when the
// subscription_context_field option is on, which table mode does not generate.
func pluginCallable(cfg *config.Config, t *config.TypeReference) bool {
	return t.Definition.Name != "Subscription" || !cfg.SubscriptionContextField
}

// collectPluginCalls returns, by name, the marshalers and unmarshalers of the functions
// mode that the Go files in the package of the executor of data refer to, called or as
// values, and how: as methods of an execution context or as functions. It leaves out the
// files of the executor itself, which table mode writes next, and the forms that the files
// declare themselves, a function of the name or a method of the name of executionContext.
// A file that does not parse is left to the compiler.
func collectPluginCalls(data *codegen.Data) (map[string]callForms, error) {
	known := map[string]bool{}
	for _, t := range data.ReferencedTypes {
		// The names are empty for the types that are not input or output types.
		if name := t.UnmarshalFunc(); name != "" {
			known[name] = true
		}
		if name := t.MarshalFunc(); name != "" && pluginCallable(data.Config, t) {
			known[name] = true
		}
	}
	exec := data.Config.Exec
	dir := exec.Dir()
	executor := map[string]bool{filepath.Clean(exec.Filename): true}
	if exec.Layout == config.ExecLayoutFollowSchema {
		// The files of the follow-schema layout are named after the sources of the types
		// and directives, with the prelude of the built-in ones.
		executor = map[string]bool{
			filepath.Join(dir, "root_.generated.go"):       true,
			filepath.Join(dir, filename(nil, data.Config)): true,
		}
		add := func(p *gqlast.Position) {
			executor[filepath.Join(dir, filename(p, data.Config))] = true
		}
		for _, src := range data.Config.Sources {
			add(&gqlast.Position{Src: src})
		}
		for _, def := range data.Config.Schema.Types {
			add(def.Position)
		}
		for _, d := range data.Config.Schema.Directives {
			add(d.Position)
		}
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	refs := &pluginRefs{
		known:    known,
		calls:    map[string]callForms{},
		declared: map[string]callForms{},
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		if executor[path] {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			refs.decl(decl)
		}
	}
	for name, declared := range refs.declared {
		forms, ok := refs.calls[name]
		if !ok {
			continue
		}
		forms.method = forms.method && !declared.method
		forms.function = forms.function && !declared.function
		if forms.method || forms.function {
			refs.calls[name] = forms
		} else {
			delete(refs.calls, name)
		}
	}
	return refs.calls, nil
}

// pluginRefs collects the references of the Go files of plugins to the marshalers and
// unmarshalers of the functions mode: see collectPluginCalls.
type pluginRefs struct {
	// known holds the names of the marshalers and unmarshalers.
	known map[string]bool
	// calls holds how the files refer to them.
	calls map[string]callForms
	// declared holds the forms that the files declare themselves.
	declared map[string]callForms
	// contexts holds the names of the parameters and receivers of the declaration being
	// read that are execution contexts, through which the files call the methods.
	contexts map[string]bool
}

// decl reads a declaration of a file.
func (r *pluginRefs) decl(decl ast.Decl) {
	r.contexts = map[string]bool{}
	if fn, ok := decl.(*ast.FuncDecl); ok {
		forms := r.declared[fn.Name.Name]
		if fn.Recv == nil {
			forms.function = true
		} else if isExecutionContext(fn.Recv.List[0].Type) {
			forms.method = true
		}
		r.declared[fn.Name.Name] = forms
	}
	// The execution contexts of the declaration: the receiver or a parameter of it or of a
	// function literal in it, and a variable declared of the type or with a new one.
	ast.Inspect(decl, func(n ast.Node) bool {
		var lists []*ast.FieldList
		switch n := n.(type) {
		case *ast.FuncDecl:
			lists = append(lists, n.Recv, n.Type.Params)
		case *ast.FuncLit:
			lists = append(lists, n.Type.Params)
		case *ast.ValueSpec:
			if n.Type != nil && isExecutionContext(n.Type) {
				for _, name := range n.Names {
					r.contexts[name.Name] = true
				}
			}
			for i, value := range n.Values {
				if i < len(n.Names) && newsExecutionContext(value) {
					r.contexts[n.Names[i].Name] = true
				}
			}
		case *ast.AssignStmt:
			for i, value := range n.Rhs {
				if i >= len(n.Lhs) || !newsExecutionContext(value) {
					continue
				}
				if id, ok := n.Lhs[i].(*ast.Ident); ok {
					r.contexts[id.Name] = true
				}
			}
		}
		for _, list := range lists {
			if list == nil {
				continue
			}
			for _, field := range list.List {
				if isExecutionContext(field.Type) {
					for _, name := range field.Names {
						r.contexts[name.Name] = true
					}
				}
			}
		}
		return true
	})
	ast.Inspect(decl, r.visit)
}

// visit records the references of a node to the marshalers and unmarshalers: a selector of
// an execution context, called or not, as a method, and any other identifier of the name
// as a function. The selector of anything else, such as a field of the tables of table
// mode, and the name of a declaration are not references.
func (r *pluginRefs) visit(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.FuncDecl:
		if n.Recv != nil {
			ast.Inspect(n.Recv, r.visit)
		}
		ast.Inspect(n.Type, r.visit)
		if n.Body != nil {
			ast.Inspect(n.Body, r.visit)
		}
		return false
	case *ast.SelectorExpr:
		if x, ok := n.X.(*ast.Ident); ok && r.contexts[x.Name] && r.known[n.Sel.Name] {
			forms := r.calls[n.Sel.Name]
			forms.method = true
			r.calls[n.Sel.Name] = forms
		}
		ast.Inspect(n.X, r.visit)
		return false
	case *ast.Ident:
		if r.known[n.Name] {
			forms := r.calls[n.Name]
			forms.function = true
			r.calls[n.Name] = forms
		}
	}
	return true
}

// newsExecutionContext reports whether the expression e makes an execution context, with
// newExecutionContext or a composite literal of executionContext or its address.
func newsExecutionContext(e ast.Expr) bool {
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
		e = u.X
	}
	switch e := e.(type) {
	case *ast.CallExpr:
		id, ok := e.Fun.(*ast.Ident)
		return ok && id.Name == "newExecutionContext"
	case *ast.CompositeLit:
		return e.Type != nil && isExecutionContext(e.Type)
	}
	return false
}

// isExecutionContext reports whether the type expression t is executionContext or a
// pointer to it, the execution context of the generated package.
func isExecutionContext(t ast.Expr) bool {
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	id, ok := t.(*ast.Ident)
	return ok && id.Name == "executionContext"
}

// PluginCall is a marshaler or an unmarshaler of the functions mode that the code that
// plugins generated calls, for which table mode generates functions with its name: see
// TablePluginCalls.
type PluginCall struct {
	// Name is the name of the marshaler or unmarshaler in the functions mode.
	Name string
	// Type is the type reference that it marshals or unmarshals.
	Type *config.TypeReference
	// Marshal is set for a marshaler, which takes the selection set and the value, and
	// unset for an unmarshaler, which takes the raw value.
	Marshal bool
	// Method is set when table mode generates it as a method of the execution context.
	Method bool
	// Function is set when table mode generates it as a function that takes the execution
	// context, which it does not generate otherwise.
	Function bool
	// Call is what the generated functions call, with the context and the execution
	// context, then the arguments of the marshaler or unmarshaler.
	Call string
}

// TablePluginCalls returns the marshalers and unmarshalers of the functions mode that the
// code that plugins generated calls, in the order of their names, with the functions that
// table mode generates for them.
func (d *Data) TablePluginCalls() []PluginCall {
	var res []PluginCall
	seen := map[string]bool{}
	add := func(name string, t *config.TypeReference, marshal bool, call func() string) {
		forms, ok := d.shared.pluginCalls[name]
		if name == "" || !ok || seen[name] {
			return
		}
		seen[name] = true
		c := call()
		u := PluginCall{
			Name:     name,
			Type:     t,
			Marshal:  marshal,
			Method:   forms.method,
			Function: forms.function && c != name,
			Call:     c,
		}
		if u.Method || u.Function {
			res = append(res, u)
		}
	}
	for _, t := range d.ReferencedTypes {
		add(t.UnmarshalFunc(), t, false, func() string { return d.UnmarshalCall(t) })
		if pluginCallable(d.Config, t) {
			add(t.MarshalFunc(), t, true, func() string { return d.MarshalCall(t) })
		}
	}
	slices.SortFunc(
		res,
		func(a, b PluginCall) int { return strings.Compare(a.Name, b.Name) },
	)
	return res
}
