package dictgen

import (
	_ "embed"
	"os"
	"path/filepath"
	"text/template"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin"
)

//go:embed roots.gotpl
var rootsTemplate string

//go:embed objects.gotpl
var objectsTemplate string

func New() plugin.Plugin {
	return &Plugin{}
}

const (
	QueryPkgName        = "gqlquery"
	MutationPkgName     = "gqlmutation"
	SubscriptionPkgName = "gqlsubscription"
	ObjectPkgName       = "gqlobject"
)

type Plugin struct{}

var _ plugin.CodeGenerator = (*Plugin)(nil)

func (m *Plugin) Name() string {
	return "dictgen"
}

func (m *Plugin) GenerateCode(data *codegen.Data) error {
	if err := generateRoots(data); err != nil {
		return err
	}

	if err := generateObjects(data); err != nil {
		return err
	}

	return nil
}

func generateRoots(data *codegen.Data) error {
	roots := map[string]*codegen.Object{
		QueryPkgName:        data.QueryRoot,
		MutationPkgName:     data.MutationRoot,
		SubscriptionPkgName: data.SubscriptionRoot,
	}

	for packageName, rootObject := range roots {
		filePath := removeAndGetDictFilepath(packageName, data)

		if rootObject == nil || len(rootObject.Fields) == 0 {
			continue
		}

		if err := templates.Render(templates.Options{
			PackageName: packageName,
			Filename:    filePath,
			Data:        rootObject,
			Funcs: template.FuncMap{
				"goName": templates.ToGoModelName,
			},
			GeneratedHeader: true,
			Template:        rootsTemplate,
			Packages:        data.Config.Packages,
		}); err != nil {
			return err
		}
	}

	return nil
}

func generateObjects(data *codegen.Data) error {
	allObjects := codegen.Objects{}
	allObjects = append(allObjects, data.Objects...)
	allObjects = append(allObjects, data.Inputs...)

	objects := codegen.Objects{}
	for _, o := range allObjects {
		if o == data.QueryRoot || o == data.MutationRoot || o == data.SubscriptionRoot {
			continue
		}

		if o.IsReserved() {
			continue
		}

		objects = append(objects, o)
	}

	filePath := removeAndGetDictFilepath(ObjectPkgName, data)

	if len(objects) != 0 {
		if err := templates.Render(templates.Options{
			PackageName: ObjectPkgName,
			Filename:    filePath,
			Data:        objects,
			Funcs: template.FuncMap{
				"goName": templates.ToGoModelName,
			},
			GeneratedHeader: true,
			Template:        objectsTemplate,
			Packages:        data.Config.Packages,
		}); err != nil {
			return err
		}
	}

	return nil
}

func removeAndGetDictFilepath(packageName string, data *codegen.Data) string {
	filePath := filepath.Join(packageName, "names.go")
	if data.Config.Exec.Layout == config.ExecLayoutFollowSchema {
		filePath = filepath.Join(data.Config.Exec.DirName, filePath)
	} else {
		filePath = filepath.Join(filepath.Dir(data.Config.Exec.Filename), filePath)
	}

	_ = os.RemoveAll(filepath.Dir(filePath))

	return filePath
}
