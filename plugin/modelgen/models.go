package modelgen

import (
	_ "embed"
	"fmt"
	"go/types"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin"
)

//go:embed models.gotpl
var modelTemplate string

type (
	BuildMutateHook = func(b *ModelBuild) *ModelBuild
	FieldMutateHook = func(td *ast.Definition, fd *ast.FieldDefinition, f *Field) (*Field, error)
)

// DefaultFieldMutateHook is the default hook for the Plugin which applies the GoFieldHook and GoTagFieldHook.
func DefaultFieldMutateHook(td *ast.Definition, fd *ast.FieldDefinition, f *Field) (*Field, error) {
	var err error
	f, err = GoFieldHook(td, fd, f)
	if err != nil {
		return f, err
	}
	return GoTagFieldHook(td, fd, f)
}

// DefaultBuildMutateHook is the default hook for the Plugin which mutate ModelBuild.
func DefaultBuildMutateHook(b *ModelBuild) *ModelBuild {
	return b
}

type ModelBuild struct {
	PackageName string
	Interfaces  []*Interface
	Models      []*Object
	Enums       []*Enum
	Scalars     []string
}

type Interface struct {
	Description string
	Name        string
	Fields      []*Field
	Implements  []string
	OmitCheck   bool
	Models      []*Object
}

type Object struct {
	Description string
	Name        string
	Fields      []*Field
	Implements  []string
}

type Field struct {
	Description string
	// Name is the field's name as it appears in the schema
	Name string
	// GoName is the field's name as it appears in the generated Go code
	GoName    string
	Type      types.Type
	Tag       string
	Omittable bool
}

type Enum struct {
	Description string
	Name        string
	Values      []*EnumValue
}

type EnumValue struct {
	Description string
	Name        string
}

func New() plugin.Plugin {
	return &Plugin{
		MutateHook: DefaultBuildMutateHook,
		FieldHook:  DefaultFieldMutateHook,
	}
}

type Plugin struct {
	MutateHook BuildMutateHook
	FieldHook  FieldMutateHook
}

var _ plugin.ConfigMutator = &Plugin{}

func (m *Plugin) Name() string {
	return "modelgen"
}

func (m *Plugin) MutateConfig(cfg *config.Config) error {
	b := &ModelBuild{
		PackageName: cfg.Model.Package,
	}

	for _, schemaType := range cfg.Schema.Types {
		if cfg.Models.UserDefined(schemaType.Name) {
			continue
		}
		switch schemaType.Kind {
		case ast.Interface, ast.Union:
			var fields []*Field
			var err error
			if !cfg.OmitGetters {
				fields, err = m.generateFields(cfg, schemaType)
				if err != nil {
					return err
				}
			}

			it := &Interface{
				Description: schemaType.Description,
				Name:        schemaType.Name,
				Implements:  schemaType.Interfaces,
				Fields:      fields,
				OmitCheck:   cfg.OmitInterfaceChecks,
			}

			b.Interfaces = append(b.Interfaces, it)
		case ast.Object, ast.InputObject:
			if schemaType == cfg.Schema.Query || schemaType == cfg.Schema.Mutation || schemaType == cfg.Schema.Subscription {
				continue
			}

			fields, err := m.generateFields(cfg, schemaType)
			if err != nil {
				return err
			}

			it := &Object{
				Description: schemaType.Description,
				Name:        schemaType.Name,
				Fields:      fields,
			}

			// If Interface A implements interface B, and Interface C also implements interface B
			// then both A and C have methods of B.
			// The reason for checking unique is to prevent the same method B from being generated twice.
			uniqueMap := map[string]bool{}
			for _, implementor := range cfg.Schema.GetImplements(schemaType) {
				if !uniqueMap[implementor.Name] {
					it.Implements = append(it.Implements, implementor.Name)
					uniqueMap[implementor.Name] = true
				}
				// for interface implements
				for _, iface := range implementor.Interfaces {
					if !uniqueMap[iface] {
						it.Implements = append(it.Implements, iface)
						uniqueMap[iface] = true
					}
				}
			}

			b.Models = append(b.Models, it)
		case ast.Enum:
			it := &Enum{
				Name:        schemaType.Name,
				Description: schemaType.Description,
			}

			for _, v := range schemaType.EnumValues {
				it.Values = append(it.Values, &EnumValue{
					Name:        v.Name,
					Description: v.Description,
				})
			}

			b.Enums = append(b.Enums, it)
		case ast.Scalar:
			b.Scalars = append(b.Scalars, schemaType.Name)
		}
	}
	sort.Slice(b.Enums, func(i, j int) bool { return b.Enums[i].Name < b.Enums[j].Name })
	sort.Slice(b.Models, func(i, j int) bool { return b.Models[i].Name < b.Models[j].Name })
	sort.Slice(b.Interfaces, func(i, j int) bool { return b.Interfaces[i].Name < b.Interfaces[j].Name })

	// if we are not just turning all struct-type fields in generated structs into pointers, we need to at least
	// check for cyclical relationships and recursive structs
	if !cfg.StructFieldsAlwaysPointers {
		findAndHandleCyclicalRelationships(b)
	}

	for _, it := range b.Enums {
		cfg.Models.Add(it.Name, cfg.Model.ImportPath()+"."+templates.ToGo(it.Name))
	}
	for _, it := range b.Models {
		cfg.Models.Add(it.Name, cfg.Model.ImportPath()+"."+templates.ToGo(it.Name))
	}
	for _, it := range b.Interfaces {
		// On a given interface we want to keep a reference to all the models that implement it
		for _, model := range b.Models {
			for _, impl := range model.Implements {
				if impl == it.Name {
					// If it does, add it to the Interface's Models
					it.Models = append(it.Models, model)
				}
			}
		}
		cfg.Models.Add(it.Name, cfg.Model.ImportPath()+"."+templates.ToGo(it.Name))
	}
	for _, it := range b.Scalars {
		cfg.Models.Add(it, "github.com/99designs/gqlgen/graphql.String")
	}

	if len(b.Models) == 0 && len(b.Enums) == 0 && len(b.Interfaces) == 0 && len(b.Scalars) == 0 {
		return nil
	}

	if m.MutateHook != nil {
		b = m.MutateHook(b)
	}

	getInterfaceByName := func(name string) *Interface {
		// Allow looking up interfaces, so template can generate getters for each field
		for _, i := range b.Interfaces {
			if i.Name == name {
				return i
			}
		}

		return nil
	}
	gettersGenerated := make(map[string]map[string]struct{})
	generateGetter := func(model *Object, field *Field) string {
		if model == nil || field == nil {
			return ""
		}

		// Let templates check if a given getter has been generated already
		typeGetters, exists := gettersGenerated[model.Name]
		if !exists {
			typeGetters = make(map[string]struct{})
			gettersGenerated[model.Name] = typeGetters
		}

		_, exists = typeGetters[field.GoName]
		typeGetters[field.GoName] = struct{}{}
		if exists {
			return ""
		}

		_, interfaceFieldTypeIsPointer := field.Type.(*types.Pointer)
		var structFieldTypeIsPointer bool
		for _, f := range model.Fields {
			if f.GoName == field.GoName {
				_, structFieldTypeIsPointer = f.Type.(*types.Pointer)
				break
			}
		}
		goType := templates.CurrentImports.LookupType(field.Type)
		if strings.HasPrefix(goType, "[]") {
			getter := fmt.Sprintf("func (this %s) Get%s() %s {\n", templates.ToGo(model.Name), field.GoName, goType)
			getter += fmt.Sprintf("\tif this.%s == nil { return nil }\n", field.GoName)
			getter += fmt.Sprintf("\tinterfaceSlice := make(%s, 0, len(this.%s))\n", goType, field.GoName)
			getter += fmt.Sprintf("\tfor _, concrete := range this.%s { interfaceSlice = append(interfaceSlice, ", field.GoName)
			if interfaceFieldTypeIsPointer && !structFieldTypeIsPointer {
				getter += "&"
			} else if !interfaceFieldTypeIsPointer && structFieldTypeIsPointer {
				getter += "*"
			}
			getter += "concrete) }\n"
			getter += "\treturn interfaceSlice\n"
			getter += "}"
			return getter
		} else {
			getter := fmt.Sprintf("func (this %s) Get%s() %s { return ", templates.ToGo(model.Name), field.GoName, goType)

			if interfaceFieldTypeIsPointer && !structFieldTypeIsPointer {
				getter += "&"
			} else if !interfaceFieldTypeIsPointer && structFieldTypeIsPointer {
				getter += "*"
			}

			getter += fmt.Sprintf("this.%s }", field.GoName)
			return getter
		}
	}
	funcMap := template.FuncMap{
		"getInterfaceByName": getInterfaceByName,
		"generateGetter":     generateGetter,
	}
	newModelTemplate := modelTemplate
	if cfg.Model.ModelTemplate != "" {
		newModelTemplate = readModelTemplate(cfg.Model.ModelTemplate)
	}

	err := templates.Render(templates.Options{
		PackageName:     cfg.Model.Package,
		Filename:        cfg.Model.Filename,
		Data:            b,
		GeneratedHeader: true,
		Packages:        cfg.Packages,
		Template:        newModelTemplate,
		Funcs:           funcMap,
	})
	if err != nil {
		return err
	}

	// We may have generated code in a package we already loaded, so we reload all packages
	// to allow packages to be compared correctly
	cfg.ReloadAllPackages()

	return nil
}

func (m *Plugin) generateFields(cfg *config.Config, schemaType *ast.Definition) ([]*Field, error) {
	binder := cfg.NewBinder()
	fields := make([]*Field, 0)

	var omittableType types.Type

	for _, field := range schemaType.Fields {
		var typ types.Type
		fieldDef := cfg.Schema.Types[field.Type.Name()]

		if cfg.Models.UserDefined(field.Type.Name()) {
			var err error
			typ, err = binder.FindTypeFromName(cfg.Models[field.Type.Name()].Model[0])
			if err != nil {
				return nil, err
			}
		} else {
			switch fieldDef.Kind {
			case ast.Scalar:
				// no user defined model, referencing a default scalar
				typ = types.NewNamed(
					types.NewTypeName(0, cfg.Model.Pkg(), "string", nil),
					nil,
					nil,
				)

			case ast.Interface, ast.Union:
				// no user defined model, referencing a generated interface type
				typ = types.NewNamed(
					types.NewTypeName(0, cfg.Model.Pkg(), templates.ToGo(field.Type.Name()), nil),
					types.NewInterfaceType([]*types.Func{}, []types.Type{}),
					nil,
				)

			case ast.Enum:
				// no user defined model, must reference a generated enum
				typ = types.NewNamed(
					types.NewTypeName(0, cfg.Model.Pkg(), templates.ToGo(field.Type.Name()), nil),
					nil,
					nil,
				)

			case ast.Object, ast.InputObject:
				// no user defined model, must reference a generated struct
				typ = types.NewNamed(
					types.NewTypeName(0, cfg.Model.Pkg(), templates.ToGo(field.Type.Name()), nil),
					types.NewStruct(nil, nil),
					nil,
				)

			default:
				panic(fmt.Errorf("unknown ast type %s", fieldDef.Kind))
			}
		}

		name := templates.ToGo(field.Name)
		if nameOveride := cfg.Models[schemaType.Name].Fields[field.Name].FieldName; nameOveride != "" {
			name = nameOveride
		}

		typ = binder.CopyModifiersFromAst(field.Type, typ)

		if cfg.StructFieldsAlwaysPointers {
			if isStruct(typ) && (fieldDef.Kind == ast.Object || fieldDef.Kind == ast.InputObject) {
				typ = types.NewPointer(typ)
			}
		}

		f := &Field{
			Name:        field.Name,
			GoName:      name,
			Type:        typ,
			Description: field.Description,
			Tag:         getStructTagFromField(cfg, field),
			Omittable:   cfg.NullableInputOmittable && schemaType.Kind == ast.InputObject && !field.Type.NonNull,
		}

		if m.FieldHook != nil {
			mf, err := m.FieldHook(schemaType, field, f)
			if err != nil {
				return nil, fmt.Errorf("generror: field %v.%v: %w", schemaType.Name, field.Name, err)
			}
			f = mf
		}

		if f.Omittable {
			if schemaType.Kind != ast.InputObject || field.Type.NonNull {
				return nil, fmt.Errorf("generror: field %v.%v: omittable is only applicable to nullable input fields", schemaType.Name, field.Name)
			}

			var err error

			if omittableType == nil {
				omittableType, err = binder.FindTypeFromName("github.com/99designs/gqlgen/graphql.Omittable")
				if err != nil {
					return nil, err
				}
			}

			f.Type, err = binder.InstantiateType(omittableType, []types.Type{f.Type})
			if err != nil {
				return nil, fmt.Errorf("generror: field %v.%v: %w", schemaType.Name, field.Name, err)
			}
		}

		fields = append(fields, f)
	}

	// appending extra fields at the end of the fields list.
	modelcfg := cfg.Models[schemaType.Name]
	if len(modelcfg.ExtraFields) > 0 {
		ff := make([]*Field, 0, len(modelcfg.ExtraFields))
		for fname, fspec := range modelcfg.ExtraFields {
			ftype := buildType(fspec.Type)

			tag := `json:"-"`
			if fspec.OverrideTags != "" {
				tag = fspec.OverrideTags
			}

			ff = append(ff,
				&Field{
					Name:        fname,
					GoName:      fname,
					Type:        ftype,
					Description: fspec.Description,
					Tag:         tag,
				})
		}

		sort.Slice(ff, func(i, j int) bool {
			return ff[i].Name < ff[j].Name
		})

		fields = append(fields, ff...)
	}

	return fields, nil
}

func getStructTagFromField(cfg *config.Config, field *ast.FieldDefinition) string {
	if !field.Type.NonNull && (cfg.EnableModelJsonOmitemptyTag == nil || *cfg.EnableModelJsonOmitemptyTag) {
		return `json:"` + field.Name + `,omitempty"`
	}
	return `json:"` + field.Name + `"`
}

// GoTagFieldHook prepends the goTag directive to the generated Field f.
// When applying the Tag to the field, the field
// name is used if no value argument is present.
func GoTagFieldHook(td *ast.Definition, fd *ast.FieldDefinition, f *Field) (*Field, error) {
	args := make([]string, 0)
	for _, goTag := range fd.Directives.ForNames("goTag") {
		key := ""
		value := fd.Name

		if arg := goTag.Arguments.ForName("key"); arg != nil {
			if k, err := arg.Value.Value(nil); err == nil {
				key = k.(string)
			}
		}

		if arg := goTag.Arguments.ForName("value"); arg != nil {
			if v, err := arg.Value.Value(nil); err == nil {
				value = v.(string)
			}
		}

		args = append(args, key+":\""+value+"\"")
	}

	if len(args) > 0 {
		f.Tag = removeDuplicateTags(f.Tag + " " + strings.Join(args, " "))
	}

	return f, nil
}

// splitTagsBySpace split tags by space, except when space is inside quotes
func splitTagsBySpace(tagsString string) []string {
	var tags []string
	var currentTag string
	inQuotes := false

	for _, c := range tagsString {
		if c == '"' {
			inQuotes = !inQuotes
		}
		if c == ' ' && !inQuotes {
			tags = append(tags, currentTag)
			currentTag = ""
		} else {
			currentTag += string(c)
		}
	}
	tags = append(tags, currentTag)

	return tags
}

// containsInvalidSpace checks if the tagsString contains invalid space
func containsInvalidSpace(valuesString string) bool {
	// get rid of quotes
	valuesString = strings.ReplaceAll(valuesString, "\"", "")
	if strings.Contains(valuesString, ",") {
		// split by comma,
		values := strings.Split(valuesString, ",")
		for _, value := range values {
			if strings.TrimSpace(value) != value {
				return true
			}
		}
		return false
	}
	if strings.Contains(valuesString, ";") {
		// split by semicolon, which is common in gorm
		values := strings.Split(valuesString, ";")
		for _, value := range values {
			if strings.TrimSpace(value) != value {
				return true
			}
		}
		return false
	}
	// single value
	if strings.TrimSpace(valuesString) != valuesString {
		return true
	}
	return false
}

func removeDuplicateTags(t string) string {
	processed := make(map[string]bool)
	tt := splitTagsBySpace(t)
	returnTags := ""

	// iterate backwards through tags so appended goTag directives are prioritized
	for i := len(tt) - 1; i >= 0; i-- {
		ti := tt[i]
		// check if ti contains ":", and not contains any empty space. if not, tag is in wrong format
		// correct example: json:"name"
		if !strings.Contains(ti, ":") {
			panic(fmt.Errorf("wrong format of tags: %s. goTag directive should be in format: @goTag(key: \"something\", value:\"value\"), ", t))
		}

		kv := strings.Split(ti, ":")
		if len(kv) == 0 || processed[kv[0]] {
			continue
		}

		key := kv[0]
		value := strings.Join(kv[1:], ":")
		processed[key] = true
		if len(returnTags) > 0 {
			returnTags = " " + returnTags
		}

		isContained := containsInvalidSpace(value)
		if isContained {
			panic(fmt.Errorf("tag value should not contain any leading or trailing spaces: %s", value))
		}

		returnTags = key + ":" + value + returnTags
	}

	return returnTags
}

// GoFieldHook applies the goField directive to the generated Field f.
func GoFieldHook(td *ast.Definition, fd *ast.FieldDefinition, f *Field) (*Field, error) {
	args := make([]string, 0)
	_ = args
	for _, goField := range fd.Directives.ForNames("goField") {
		if arg := goField.Arguments.ForName("name"); arg != nil {
			if k, err := arg.Value.Value(nil); err == nil {
				f.GoName = k.(string)
			}
		}

		if arg := goField.Arguments.ForName("omittable"); arg != nil {
			if k, err := arg.Value.Value(nil); err == nil {
				f.Omittable = k.(bool)
			}
		}
	}
	return f, nil
}

func isStruct(t types.Type) bool {
	_, is := t.Underlying().(*types.Struct)
	return is
}

// findAndHandleCyclicalRelationships checks for cyclical relationships between generated structs and replaces them
// with pointers. These relationships will produce compilation errors if they are not pointers.
// Also handles recursive structs.
func findAndHandleCyclicalRelationships(b *ModelBuild) {
	for ii, structA := range b.Models {
		for _, fieldA := range structA.Fields {
			if strings.Contains(fieldA.Type.String(), "NotCyclicalA") {
				fmt.Print()
			}
			if !isStruct(fieldA.Type) {
				continue
			}

			// the field Type string will be in the form "github.com/99designs/gqlgen/codegen/testserver/followschema.LoopA"
			// we only want the part after the last dot: "LoopA"
			// this could lead to false positives, as we are only checking the name of the struct type, but these
			// should be extremely rare, if it is even possible at all.
			fieldAStructNameParts := strings.Split(fieldA.Type.String(), ".")
			fieldAStructName := fieldAStructNameParts[len(fieldAStructNameParts)-1]

			// find this struct type amongst the generated structs
			for jj, structB := range b.Models {
				if structB.Name != fieldAStructName {
					continue
				}

				// check if structB contains a cyclical reference back to structA
				var cyclicalReferenceFound bool
				for _, fieldB := range structB.Fields {
					if !isStruct(fieldB.Type) {
						continue
					}

					fieldBStructNameParts := strings.Split(fieldB.Type.String(), ".")
					fieldBStructName := fieldBStructNameParts[len(fieldBStructNameParts)-1]
					if fieldBStructName == structA.Name {
						cyclicalReferenceFound = true
						fieldB.Type = types.NewPointer(fieldB.Type)
						// keep looping in case this struct has additional fields of this type
					}
				}

				// if this is a recursive struct (i.e. structA == structB), ensure that we only change this field to a pointer once
				if cyclicalReferenceFound && ii != jj {
					fieldA.Type = types.NewPointer(fieldA.Type)
					break
				}
			}
		}
	}
}

func readModelTemplate(customModelTemplate string) string {
	contentBytes, err := os.ReadFile(customModelTemplate)
	if err != nil {
		panic(err)
	}
	return string(contentBytes)
}
