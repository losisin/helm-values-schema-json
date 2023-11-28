package pkg

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type propertySuite struct{}

var _ = Suite(&propertySuite{})

type ExampleJSONBasic struct {
	Omitted    string  `json:"-,omitempty"`
	Bool       bool    `json:",omitempty"`
	Integer    int     `json:",omitempty"`
	Integer8   int8    `json:",omitempty"`
	Integer16  int16   `json:",omitempty"`
	Integer32  int32   `json:",omitempty"`
	Integer64  int64   `json:",omitempty"`
	UInteger   uint    `json:",omitempty"`
	UInteger8  uint8   `json:",omitempty"`
	UInteger16 uint16  `json:",omitempty"`
	UInteger32 uint32  `json:",omitempty"`
	UInteger64 uint64  `json:",omitempty"`
	String     string  `json:",omitempty"`
	Bytes      []byte  `json:",omitempty"`
	Float32    float32 `json:",omitempty"`
	Float64    float64
	Timestamp  time.Time `json:",omitempty"`
}

func (s *propertySuite) TestLoad(c *C) {
	j := &Document{}
	j.Read(&ExampleJSONBasic{})

	c.Assert(*j, DeepEquals, Document{
		Schema: "http://json-schema.org/schema#",
		property: property{
			Type:     "object",
			Required: []string{"Float64"},
			Properties: map[string]*property{
				"Bool":       {Type: "boolean"},
				"Integer":    {Type: "integer"},
				"Integer8":   {Type: "integer"},
				"Integer16":  {Type: "integer"},
				"Integer32":  {Type: "integer"},
				"Integer64":  {Type: "integer"},
				"UInteger":   {Type: "integer"},
				"UInteger8":  {Type: "integer"},
				"UInteger16": {Type: "integer"},
				"UInteger32": {Type: "integer"},
				"UInteger64": {Type: "integer"},
				"String":     {Type: "string"},
				"Bytes":      {Type: "string"},
				"Float32":    {Type: "number"},
				"Float64":    {Type: "number"},
				"Timestamp":  {Type: "string", Format: "date-time"},
			},
		},
	})
}

type SliceStruct struct {
	Value string
}

type ExampleJSONBasicSlices struct {
	Slice            []string      `json:",foo,omitempty"`
	SliceOfInterface []interface{} `json:",foo"`
	SliceOfStruct    []SliceStruct
}

func (s *propertySuite) TestLoadSliceAndContains(c *C) {
	j := &Document{}
	j.Read(&ExampleJSONBasicSlices{})

	c.Assert(*j, DeepEquals, Document{
		Schema: "http://json-schema.org/schema#",
		property: property{
			Type: "object",
			Properties: map[string]*property{
				"Slice": {
					Type:  "array",
					Items: &property{Type: "string"},
				},
				"SliceOfInterface": {
					Type: "array",
				},
				"SliceOfStruct": {
					Type: "array",
					Items: &property{
						Type:     "object",
						Required: []string{"Value"},
						Properties: map[string]*property{
							"Value": {
								Type: "string",
							},
						},
					},
				},
			},

			Required: []string{"SliceOfInterface", "SliceOfStruct"},
		},
	})
}

type ExampleJSONNestedStruct struct {
	Struct struct {
		Foo string
	}
}

func (s *propertySuite) TestString(c *C) {
	j := &Document{}
	j.Read(true)

	expected := "{\n" +
		"    \"$schema\": \"http://json-schema.org/schema#\",\n" +
		"    \"type\": \"boolean\"\n" +
		"}"

	c.Assert(j.String(), Equals, expected)
}

type ExampleJSONBasicMaps struct {
	Maps           map[string]string `json:",omitempty"`
	MapOfInterface map[string]interface{}
}

func TestLoadMapDeep(t *testing.T) {
	t.Run("within a struct map of string to string", func(t *testing.T) {
		j := &Document{}
		j.Read(&ExampleJSONBasicMaps{
			Maps: map[string]string{
				"aString":          "ok1",
				"anotherString":    "anotherValue",
				"yetAnotherString": "anotherValue",
			},
		})

		expected := Document{
			Schema: "http://json-schema.org/schema#",
			property: property{
				Type: "object",
				Properties: map[string]*property{
					"Maps": {
						Type: "object",
						Properties: map[string]*property{
							"aString":          {Type: "string"},
							"anotherString":    {Type: "string"},
							"yetAnotherString": {Type: "string"},
						},
					},
					"MapOfInterface": {
						Type: "object",
					},
				},
				Required: []string{"MapOfInterface"},
			},
		}
		if !cmp.Equal(expected, *j, cmp.AllowUnexported(Document{})) {
			t.Fail()
			fmt.Println(cmp.Diff(expected, *j, cmp.AllowUnexported(Document{})))
		}
	})
	t.Run("map of string to string", func(t *testing.T) {
		j := &Document{}
		j.Read(map[string]string{
			"aString":          "ok1",
			"anotherString":    "anotherValue",
			"yetAnotherString": "anotherValue",
		})

		expected := Document{
			Schema: "http://json-schema.org/schema#",
			property: property{
				Type: "object",
				Properties: map[string]*property{
					"aString":          {Type: "string"},
					"anotherString":    {Type: "string"},
					"yetAnotherString": {Type: "string"},
				},
			},
		}
		if !cmp.Equal(expected, *j, cmp.AllowUnexported(Document{})) {
			t.Fail()
			fmt.Println(cmp.Diff(expected, *j, cmp.AllowUnexported(Document{})))
		}
	})
	t.Run("map of string to interface", func(t *testing.T) {
		j := &Document{}
		j.Read(map[string]interface{}{
			"aString":          "ok1",
			"anotherString":    "anotherValue",
			"yetAnotherString": "anotherValue",
			"aStringInsideMap": "ok2",
			"aBool":            true,
			"anInt":            1,
			"aFloat":           1.699,
			"sliceOfString":    []string{"something"},
			"aMapOfStringToString": map[string]string{
				"justAString": "ok3",
			},
			"aMapOfStringToInterface": map[string]interface{}{
				"justAnotherString": "ok4",
				"anotherBool":       true,
				"anotherInt":        1,
				"anotherFloat":      1.699,
			},
			"aMapOfInterfaceToInterface": map[interface{}]interface{}{
				"justAnotherString": "ok4",
				"anotherBool":       true,
				"anotherInt":        1,
				"anotherFloat":      1.699,
				"emptySliceOfFloat": []float64{},
			},
			"aMapOfInterfaceToMapOfInterfaceToInterface": map[interface{}]interface{}{
				"aPointerToMapOfInterfaceToInterface": &map[interface{}]interface{}{
					"justAnotherString":     "ok4",
					"anotherBool":           true,
					"anotherInt":            1,
					"anotherFloat":          1.699,
					"nilData":               nil,
					"zeroIntValue":          0,
					"zeroStringValue":       "",
					"sliceOfInt":            []int{1},
					"emptySliceOfInterface": []interface{}{},
				},
			},
		})

		expected := Document{
			Schema: "http://json-schema.org/schema#",
			property: property{
				Type: "object",
				Properties: map[string]*property{
					"aString":          {Type: "string"},
					"anotherString":    {Type: "string"},
					"yetAnotherString": {Type: "string"},
					"aStringInsideMap": {Type: "string"},
					"aBool":            {Type: "boolean"},
					"anInt":            {Type: "integer"},
					"aFloat":           {Type: "number"},
					"sliceOfString":    {Type: "array", Items: &property{Type: "string"}},
					"aMapOfStringToString": {
						Type:       "object",
						Properties: map[string]*property{"justAString": {Type: "string"}},
					},
					"aMapOfStringToInterface": {
						Type: "object",
						Properties: map[string]*property{
							"anotherBool":       {Type: "boolean"},
							"anotherFloat":      {Type: "number"},
							"anotherInt":        {Type: "integer"},
							"justAnotherString": {Type: "string"},
						},
					},
					"aMapOfInterfaceToInterface": {
						Type: "object",
						Properties: map[string]*property{
							"anotherBool":       {Type: "boolean"},
							"anotherFloat":      {Type: "number"},
							"anotherInt":        {Type: "integer"},
							"emptySliceOfFloat": {Type: "array", Items: &property{Type: "number"}},
							"justAnotherString": {Type: "string"},
						},
					},
					"aMapOfInterfaceToMapOfInterfaceToInterface": {
						Type: "object",
						Properties: map[string]*property{
							"aPointerToMapOfInterfaceToInterface": {
								Type: "object",
								Properties: map[string]*property{
									"anotherBool":           {Type: "boolean"},
									"anotherFloat":          {Type: "number"},
									"anotherInt":            {Type: "integer"},
									"emptySliceOfInterface": {Type: "array"},
									"justAnotherString":     {Type: "string"},
									"nilData":               {Type: "null"},
									"sliceOfInt":            {Type: "array", Items: &property{Type: "integer"}},
									"zeroIntValue":          {Type: "integer"},
									"zeroStringValue":       {Type: "string"},
								},
							},
						},
					},
				},
			},
		}
		if !cmp.Equal(expected, *j, cmp.AllowUnexported(Document{})) {
			t.Fail()
			fmt.Println(cmp.Diff(expected, *j, cmp.AllowUnexported(Document{})))
		}
	})
	t.Run("slice of interface with string value", func(t *testing.T) {
		j := &Document{}
		j.Read(map[string]interface{}{
			"sliceOfInterfaceWithString": []interface{}{"something"},
		})

		expected := Document{
			Schema: "http://json-schema.org/schema#",
			property: property{
				Type: "object",
				Properties: map[string]*property{
					"sliceOfInterfaceWithString": {
						Type: "array",
						Items: &property{
							Type: "string",
						},
					},
				},
			},
		}
		if !cmp.Equal(expected, *j, cmp.AllowUnexported(Document{})) {
			t.Fail()
			fmt.Println(cmp.Diff(expected, *j, cmp.AllowUnexported(Document{})))
		}
	})
	t.Run("slice of interface with int value", func(t *testing.T) {
		j := &Document{}
		j.Read(map[string]interface{}{
			"sliceOfInterfaceWithInt": []interface{}{1},
		})

		expected := Document{
			Schema: "http://json-schema.org/schema#",
			property: property{
				Type: "object",
				Properties: map[string]*property{
					"sliceOfInterfaceWithInt": {
						Type: "array",
						Items: &property{
							Type: "integer",
						},
					},
				},
			},
		}
		if !cmp.Equal(expected, *j, cmp.AllowUnexported(Document{})) {
			t.Fail()
			fmt.Println(cmp.Diff(expected, *j, cmp.AllowUnexported(Document{})))
		}
	})
	t.Run("slice of interface with float value", func(t *testing.T) {
		j := &Document{}
		j.Read(map[string]interface{}{
			"sliceOfInterfaceWithFloat": []interface{}{1.555},
		})

		expected := Document{
			Schema: "http://json-schema.org/schema#",
			property: property{
				Type: "object",
				Properties: map[string]*property{
					"sliceOfInterfaceWithFloat": {
						Type: "array",
						Items: &property{
							Type: "number",
						},
					},
				},
			},
		}
		if !cmp.Equal(expected, *j, cmp.AllowUnexported(Document{})) {
			t.Fail()
			fmt.Println(cmp.Diff(expected, *j, cmp.AllowUnexported(Document{})))
		}
	})
	t.Run("slice of interface with map value", func(t *testing.T) {
		j := &Document{}
		j.Read(map[string]interface{}{
			"sliceOfInterfaceWithMapValue": []interface{}{
				map[interface{}]interface{}{
					"someString":       "another",
					"someStringSlice":  []string{"hmm"},
					"emptyStringSlice": []string{},
					"intValue":         1,
					"floatValue":       1.55,
				},
			},
		})

		expected := Document{
			Schema: "http://json-schema.org/schema#",
			property: property{
				Type: "object",
				Properties: map[string]*property{
					"sliceOfInterfaceWithMapValue": {
						Type: "array",
						Items: &property{
							Type: "object",
							Properties: map[string]*property{
								"emptyStringSlice": {Type: "array", Items: &property{Type: "string"}},
								"floatValue":       {Type: "number"},
								"intValue":         {Type: "integer"},
								"someString":       {Type: "string"},
								"someStringSlice":  {Type: "array", Items: &property{Type: "string"}},
							},
						},
					},
				},
			},
		}
		if !cmp.Equal(expected, *j, cmp.AllowUnexported(Document{})) {
			t.Fail()
			fmt.Println(cmp.Diff(expected, *j, cmp.AllowUnexported(Document{})))
		}
	})
}
