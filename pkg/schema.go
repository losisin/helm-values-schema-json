package pkg

import (
	"encoding/json"
	"reflect"
	"strings"
)

var defaultSchema = "http://json-schema.org/schema#"

type Document struct {
	Schema string `json:"$schema,omitempty"`
	property
}

// NewDocument creates a new JSON-Schema Document with the specified schema.
func NewDocument(schema string) *Document {
	return &Document{
		Schema: schema,
	}
}

// Reads the variable structure into the JSON-Schema Document
func (d *Document) Read(variable interface{}) {
	d.setDefaultSchema()

	value := reflect.ValueOf(variable)
	d.read(value, "")
}

func (d *Document) setDefaultSchema() {
	if d.Schema == "" {
		d.Schema = defaultSchema
	}
}

// Marshal returns the JSON encoding of the Document
func (d *Document) Marshal() ([]byte, error) {
	return json.MarshalIndent(d, "", "    ")
}

// String return the JSON encoding of the Document as a string
func (d *Document) String() string {
	jsonBytes, _ := d.Marshal()
	return string(jsonBytes)
}

type property struct {
	Type                 string               `json:"type,omitempty"`
	Format               string               `json:"format,omitempty"`
	Items                *property            `json:"items,omitempty"`
	Properties           map[string]*property `json:"properties,omitempty"`
	Required             []string             `json:"required,omitempty"`
	AdditionalProperties bool                 `json:"additionalProperties,omitempty"`
}

func (p *property) read(v reflect.Value, opts tagOptions) {
	if !v.IsValid() {
		p.Type = "null"
		return
	}
	jsType, format, kind := getTypeFromMapping(v.Type())
	if jsType != "" {
		p.Type = jsType
	}
	if format != "" {
		p.Format = format
	}

	switch kind {
	case reflect.Slice:
		p.readFromSlice(v)
	case reflect.Map:
		p.readFromMap(v)
	case reflect.Struct:
		p.readFromStruct(v)
	case reflect.Ptr, reflect.Interface:
		p.read(v.Elem(), opts)
	}
}

func (p *property) readFromSlice(v reflect.Value) {
	if v.Len() == 0 {
		t := v.Type()
		jsType, _, kind := getTypeFromMapping(t.Elem())
		if kind == reflect.Uint8 {
			p.Type = "string"
		} else if jsType != "" {
			p.Items = &property{}
			if v.Len() == 0 {
				p.Items.read(reflect.Zero(t.Elem()), "")
				return
			}
			p.Items.read(v.Index(0), "")
		}
		return
	}

	_, _, kind := getTypeFromMapping(v.Index(0).Type())
	if kind == reflect.Uint8 {
		p.Type = "string"
	} else {
		p.Items = &property{}
		p.Items.read(v.Index(0), "")
	}
}

func (p *property) readFromMap(v reflect.Value) {
	properties := make(map[string]*property)
	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		keyName := mapKeyToString(key)
		properties[keyName] = &property{}
		properties[keyName].read(value, "")
	}

	if len(properties) > 0 {
		p.Properties = properties
	}
}

func mapKeyToString(key reflect.Value) string {
	keyKind := key.Kind()

	if keyKind == reflect.Interface {
		return mapKeyToString(key.Elem())
	}

	return key.String()
}

func (p *property) readFromStruct(v reflect.Value) {
	t := v.Type()
	p.Type = "object"
	p.Properties = make(map[string]*property, 0)
	p.AdditionalProperties = false

	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)

		tag := field.Tag.Get("json")
		name, opts := parseTag(tag)
		if name == "" {
			name = field.Name
		}
		if name == "-" {
			continue
		}

		p.Properties[name] = &property{}
		p.Properties[name].read(v.Field(i), opts)

		if !opts.Contains("omitempty") {
			p.Required = append(p.Required, name)
		}
	}
}

var formatMapping = map[string][]string{
	"time.Time": {"string", "date-time"},
}

var kindMapping = map[reflect.Kind]string{
	reflect.Bool:    "boolean",
	reflect.Int:     "integer",
	reflect.Int8:    "integer",
	reflect.Int16:   "integer",
	reflect.Int32:   "integer",
	reflect.Int64:   "integer",
	reflect.Uint:    "integer",
	reflect.Uint8:   "integer",
	reflect.Uint16:  "integer",
	reflect.Uint32:  "integer",
	reflect.Uint64:  "integer",
	reflect.Float32: "number",
	reflect.Float64: "number",
	reflect.String:  "string",
	reflect.Slice:   "array",
	reflect.Struct:  "object",
	reflect.Map:     "object",
}

func getTypeFromMapping(t reflect.Type) (string, string, reflect.Kind) {
	if v, ok := formatMapping[t.String()]; ok {
		return v[0], v[1], reflect.String
	}

	kind := t.Kind()
	if v, ok := kindMapping[kind]; ok {
		return v, "", kind
	}

	return "", "", kind
}

type tagOptions string

func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, ""
}

func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}

	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}
