package jsonschema

import (
	"reflect"

	"github.com/piyushsingariya/kaku/jsonschema/generator"
	"github.com/piyushsingariya/kaku/jsonschema/schema"
)

// Reflector is the main jsonschemagen command.
type Reflector struct {
	opts           generator.Options
	includeTests   bool
	inlineDefs     bool
	gen            *generator.JSONSchemaGenerator
	suppressXAttrs bool
}

func Reflect(v interface{}) (schema.JSONSchema, error) {
	r := Reflector{}
	basePackage := r.GetPackageName(v)
	typeOf := reflect.TypeOf(v)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}

	opts := generator.NewOptions()
	opts.AutoCreateDefs = !r.inlineDefs
	opts.IncludeTests = r.includeTests
	opts.SupressXAttrs = r.suppressXAttrs
	opts.LogLevel = generator.VerboseLevel

	r.opts = opts
	r.gen = generator.NewJSONSchemaGenerator(basePackage, typeOf.Name(), opts)

	return r.gen.Generate()
}

func (r *Reflector) GetPackageName(s interface{}) string {
	t := reflect.TypeOf(s)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	pkgPath := t.PkgPath()

	return pkgPath
}
