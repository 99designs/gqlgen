// Package tablecodegen generates the executor of table mode, which the graphql/tableimpl runtime
// executes from tables, from the data that the codegen package builds for the functions
// mode. The api package calls Build after codegen.BuildData, and GenerateCode in place of
// codegen.GenerateCode, when gqlgen is built with the gqlgen_table tag: only the file of
// the api package with that tag imports this package.
//
// The generator renders its own templates. Those it shares with the functions mode are
// copies of the templates of the codegen package, which call the methods of Data where
// table mode differs, and table_.gotpl defines the parts that differ in them. Some small
// functions of the codegen package are copied as well. TestCopies fails when an original
// changes, so that the copy follows it.
package tablecodegen

import (
	"embed"
	"go/types"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen"
)

// The templates of table mode, which differ from those of the functions mode in the files
// of the objects, the inputs and the fields, and in table_.gotpl.

//go:embed api!.gotpl internal!.gotpl directives_.gotpl table_.gotpl
var rootTemplateFS embed.FS

//go:embed args.gotpl field_.gotpl field_table.gotpl input_.gotpl input_table.gotpl interface.gotpl object_table.gotpl type.gotpl generated!.gotpl directives_.gotpl table_.gotpl
var schemaTemplateFS embed.FS

//go:embed api!.gotpl internal!.gotpl args.gotpl field_.gotpl field_table.gotpl input_.gotpl input_table.gotpl interface.gotpl object_table.gotpl type.gotpl generated!.gotpl directives_.gotpl table_.gotpl
var singleFileTemplateFS embed.FS

// Data is the data of the code that table mode generates: the codegen.Data of the functions
// mode, with what table mode derives from it. The templates of table mode execute with a
// Data, whose methods replace some of codegen.Data.
type Data struct {
	*codegen.Data
	// TableLinkedSchema is the schema in SDL that the tables are linked to, when it is not
	// the schema of the sources. See tableLinkedSchema.
	TableLinkedSchema string

	shared *shared
	// info is what table mode derives from the whole schema, which the files of the
	// follow-schema layout share. See tableInfo.
	info *tableInfo
	// groups are the groups of tables that the generated file of this Data builds: those
	// of its own types for a file of the follow-schema layout, and all of them for the
	// root file and the single-file layout. See TableInitGroups.
	groups [][]*codegen.Object
}

// shared is what table mode keeps of the data for every file it generates.
type shared struct {
	// built is the tableState of the data when Build returned.
	built map[string]string
	// calledByPlugins holds the marshalers and unmarshalers that the code that plugins
	// generated calls. See addPluginCall.
	calledByPlugins map[string]bool
	// pluginCalls holds, by name, the marshalers and unmarshalers of the functions mode that
	// the code that plugins generated calls, and how it calls them. See collectPluginCalls.
	pluginCalls map[string]callForms
	// funcs holds the names of the functions that table mode generates for the objects and
	// inputs of the data when Build returned. See tableFuncNames.
	funcs map[*codegen.Object]string
	// goTypes holds the Go types of the vars that the fields of the objects are bound to,
	// or of the results of their methods, which may not be the types that their
	// TypeReference marshals, as BuildData bound them. See tableGoTypes.
	goTypes map[*codegen.Field]types.Type
}

// Build does in table mode what codegen.BuildData does for it, once data holds the data of
// the functions mode: it checks the directives, names the functions of the objects and
// inputs, writes the schema that the tables are linked to where it is not that of the
// sources, and keeps the state of the data that plugins must not change.
func Build(data *codegen.Data) (*Data, error) {
	cfg := data.Config
	d := &Data{
		Data: data,
		shared: &shared{
			funcs:   map[*codegen.Object]string{},
			goTypes: tableGoTypes(data.Objects),
		},
	}
	if err := d.tableCheckDirectives(); err != nil {
		return nil, err
	}
	tableFuncNames(cfg.Schema, data.Objects, d.shared.funcs)
	tableFuncNames(cfg.Schema, data.Inputs, d.shared.funcs)
	sources, err := codegen.SerializeTransformedSchema(cfg.Schema, cfg.Sources)
	if err != nil {
		return nil, err
	}
	skipRuntime := func(name string) bool { return cfg.Directives[name].SkipRuntime }
	if d.TableLinkedSchema, err = tableLinkedSchema(
		cfg.Schema,
		tableSources(sources, data.AugmentedSources, cfg.Exec.Dir()),
		skipRuntime,
	); err != nil {
		return nil, err
	}
	d.shared.built = d.tableState()
	return d, nil
}

// tableGoTypes returns the Go types of the vars that the fields of objects are bound to,
// or of the results of their methods: those of the fields that BuildData bound to a var
// or a method of the Go type of their object.
func tableGoTypes(objects codegen.Objects) map[*codegen.Field]types.Type {
	res := map[*codegen.Field]types.Type{}
	for _, o := range objects {
		if o.Type == nil {
			continue
		}
		for _, f := range o.Fields {
			if f.IsResolver || f.GoReceiverName != "obj" || (!f.IsVariable() && !f.IsMethod()) {
				continue
			}
			switch target := lookupFieldOrMethod(o.Type, f.GoFieldName).(type) {
			case *types.Var:
				res[f] = target.Type()
			case *types.Func:
				if sig, ok := target.Type().(*types.Signature); ok && sig.Results().Len() > 0 {
					res[f] = sig.Results().At(0).Type()
				}
			}
		}
	}
	return res
}

// lookupFieldOrMethod returns the field or method called name of t, which BuildData binds
// a field to.
func lookupFieldOrMethod(t types.Type, name string) types.Object {
	obj, _, _ := types.LookupFieldOrMethod(t, true, nil, name)
	return obj
}

// FunctionSyntax reports that the generated functions take the execution context as a
// parameter: table mode always uses it, so that the type functions and the combinators
// that replace some of them are called alike. FuncReceiver, ECFuncParam, ECDot and ECArg
// write it in place of those of codegen.Data, which follow the config.
func (d *Data) FunctionSyntax() bool {
	return true
}

// FuncReceiver is codegen.Data.FuncReceiver with FunctionSyntax: no receiver.
func (d *Data) FuncReceiver() string {
	return ""
}

// ECFuncParam is codegen.Data.ECFuncParam with FunctionSyntax.
func (d *Data) ECFuncParam() string {
	return "ec *executionContext, "
}

// ECDot is codegen.Data.ECDot with FunctionSyntax: nothing.
func (d *Data) ECDot() string {
	return ""
}

// ECArg is codegen.Data.ECArg with FunctionSyntax.
func (d *Data) ECArg() string {
	return "ec, "
}

// ImplDirectivesContext is codegen.ImplDirectivesContext with the Data of table mode, whose
// methods the implDirectives template calls.
type ImplDirectivesContext struct {
	codegen.ImplDirectivesContext
	Data *Data
}

// ImplDirectivesCtx is codegen.Data.ImplDirectivesCtx with d.
func (d *Data) ImplDirectivesCtx(f codegen.ImplDirectivesField) ImplDirectivesContext {
	return ImplDirectivesContext{ImplDirectivesContext: d.Data.ImplDirectivesCtx(f), Data: d}
}

// QueryDirectivesContext is codegen.QueryDirectivesContext with the Data of table mode,
// whose methods the queryDirectives template calls.
type QueryDirectivesContext struct {
	codegen.QueryDirectivesContext
	Data *Data
}

// QueryDirectivesCtx is codegen.Data.QueryDirectivesCtx with d.
func (d *Data) QueryDirectivesCtx(dl codegen.DirectiveList) QueryDirectivesContext {
	return QueryDirectivesContext{QueryDirectivesContext: d.Data.QueryDirectivesCtx(dl), Data: d}
}

// copyData returns a copy of data that computes which objects resolve a field through a
// batch resolver from its objects now, which data may have computed before a plugin
// changed them.
func copyData(data *codegen.Data) *codegen.Data {
	c := &codegen.Data{Objects: data.Objects}
	c.HasBatchResolverFields()
	c.Config = data.Config
	c.Schema = data.Schema
	c.AllDirectives = data.AllDirectives
	c.Inputs = data.Inputs
	c.Interfaces = data.Interfaces
	c.ReferencedTypes = data.ReferencedTypes
	c.ComplexityRoots = data.ComplexityRoots
	c.QueryRoot = data.QueryRoot
	c.MutationRoot = data.MutationRoot
	c.SubscriptionRoot = data.SubscriptionRoot
	c.AugmentedSources = data.AugmentedSources
	c.Plugins = data.Plugins
	return c
}

// Args is codegen.Data.Args in table mode, which keeps the arguments of the fields in the
// tables of the objects, and unmarshals the arguments of the directives in the schema with
// their tableimpl.DirectiveDef: it keeps the functions of the arguments of the directives of
// queries.
func (d *Data) Args() map[string][]*codegen.FieldArgument {
	ret := map[string][]*codegen.FieldArgument{}
	for _, directive := range d.Directives() {
		if len(directive.Args) > 0 && directive.IsLocation(
			ast.LocationQuery,
			ast.LocationMutation,
			ast.LocationSubscription,
			ast.LocationField,
		) {
			ret[directive.ArgsFunc()] = directive.Args
		}
	}
	return ret
}

// collectBatchFieldTypes returns the names of the objects that resolve at least one field
// through a batch resolver.
//
// Root objects are excluded: a batch resolver groups sibling parents, and the root has no
// parent to group — ShortBatchResolverDeclaration returns "" for root fields.
func collectBatchFieldTypes(objects codegen.Objects) map[string]bool {
	names := make(map[string]bool)
	for _, obj := range objects {
		if obj.Root {
			continue
		}
		for _, field := range obj.Fields {
			if field.IsBatch() {
				names[obj.Name] = true
				break
			}
		}
	}
	return names
}

// batchImplementors is codegen.Data.BatchImplementors for the interfaces and the objects
// with batch fields of a Data.
func batchImplementors(
	interfaces map[string]*codegen.Interface,
	batchFieldTypes map[string]bool,
	name string,
) []codegen.InterfaceImplementor {
	iface := interfaces[name]
	if iface == nil {
		return nil
	}
	var impls []codegen.InterfaceImplementor
	for _, impl := range iface.Implementors {
		if batchFieldTypes[impl.Name] {
			impls = append(impls, impl)
		}
	}
	return impls
}
