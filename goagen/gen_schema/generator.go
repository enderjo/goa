package genschema

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/goadesign/goa/design"
	"github.com/goadesign/goa/goagen/codegen"
	"github.com/goadesign/goa/goagen/utils"

	"gopkg.in/alecthomas/kingpin.v2"
)

// Generator is the application code generator.
type Generator struct {
	genfiles []string
}

// Generate is the generator entry point called by the meta generator.
func Generate(api *design.APIDefinition) ([]string, error) {
	g, err := NewGenerator()
	if err != nil {
		return nil, err
	}
	return g.Generate(api)
}

// NewGenerator returns the application code generator.
func NewGenerator() (*Generator, error) {
	app := kingpin.New("Main generator", "application JSON schema generator")
	codegen.RegisterFlags(app)
	NewCommand().RegisterFlags(app)
	_, err := app.Parse(os.Args[1:])
	if err != nil {
		return nil, fmt.Errorf(`invalid command line: %s. Command line was "%s"`,
			err, strings.Join(os.Args, " "))
	}
	return new(Generator), nil
}

// JSONSchemaDir is the path to the directory where the schema controller is generated.
func JSONSchemaDir() string {
	return filepath.Join(codegen.OutputDir, "schema")
}

// Generate produces the skeleton main.
func (g *Generator) Generate(api *design.APIDefinition) (_ []string, err error) {
	go utils.Catch(nil, func() { g.Cleanup() })

	defer func() {
		if err != nil {
			g.Cleanup()
		}
	}()

	os.RemoveAll(JSONSchemaDir())
	os.MkdirAll(JSONSchemaDir(), 0755)
	g.genfiles = append(g.genfiles, JSONSchemaDir())
	s := APISchema(api)
	js, err := s.JSON()
	if err != nil {
		return
	}

	schemaFile := filepath.Join(JSONSchemaDir(), "schema.json")
	if err = ioutil.WriteFile(schemaFile, js, 0644); err != nil {
		return
	}
	g.genfiles = append(g.genfiles, schemaFile)

	controllerFile := filepath.Join(JSONSchemaDir(), "schema.go")
	file, err := codegen.SourceFileFor(controllerFile)
	if err != nil {
		return
	}
	imports := []*codegen.ImportSpec{
		codegen.SimpleImport("github.com/julienschmidt/httprouter"),
		codegen.SimpleImport("github.com/goadesign/goa"),
	}
	g.genfiles = append(g.genfiles, controllerFile)
	file.WriteHeader(fmt.Sprintf("%s JSON Hyper-schema", api.Name), "schema", imports)
	file.Write([]byte(jsonSchemaCtrl))
	if err = file.FormatCode(); err != nil {
		return
	}

	return g.genfiles, nil
}

// Cleanup removes all the files generated by this generator during the last invokation of Generate.
func (g *Generator) Cleanup() {
	for _, f := range g.genfiles {
		os.Remove(f)
	}
	g.genfiles = nil
}

const jsonSchemaCtrl = `
// MountController mounts the API JSON schema controller under "/schema.json".
func MountController(service goa.Service) {
	service.ServeFiles("/schema.json", "schema/schema.json")
}
`
