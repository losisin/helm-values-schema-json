// Package testdata contains files used by tests.
//
// To regenerate the test data output files, run the following:
//
//	go generate ./testdata
package testdata

//go:generate go run .. --input anchors.yaml --output anchors.schema.json
//go:generate go run .. --input basic.yaml --output basic.schema.json
//go:generate go run .. --input full.yaml --output full.schema.json --schemaRoot.id https://example.com/schema --schemaRoot.ref schema/product.json --schemaRoot.title "Helm Values Schema" --schemaRoot.description "Schema for Helm values" --schemaRoot.additionalProperties=true
//go:generate go run .. --input k8sRef.yaml --output k8sRef.schema.json --k8sSchemaVersion v1.33.1
//go:generate go run .. --input meta.yaml --output meta.schema.json
//go:generate go run .. --input noAdditionalProperties.yaml --output noAdditionalProperties.schema.json --noAdditionalProperties=true
//go:generate go run .. --input ref.yaml --output ref-draft2020.schema.json --draft 2020
//go:generate go run .. --input ref.yaml --output ref-draft7.schema.json --draft 7
//go:generate go run .. --input subschema.yaml --output subschema.schema.json

//go:generate go run .. --bundle=true --input bundle/fragment.yaml --output bundle/fragment.schema.json
//go:generate go run .. --bundle=true --input bundle/fragment.yaml --output bundle/fragment-without-id.schema.json --bundleWithoutID=true
//go:generate go run .. --bundle=true --input bundle/namecollision.yaml --output bundle/namecollision.schema.json
//go:generate go run .. --bundle=true --input bundle/nested.yaml --output bundle/nested.schema.json
//go:generate go run .. --bundle=true --input bundle/nested.yaml --output bundle/nested-without-id.schema.json --bundleWithoutID=true
//go:generate go run .. --bundle=true --input bundle/simple.yaml --output bundle/simple.schema.json
//go:generate go run .. --bundle=false --input bundle/simple.yaml --output bundle/simple-disabled.schema.json
//go:generate go run .. --bundle=true --input bundle/simple.yaml --output bundle/simple-without-id.schema.json --bundleWithoutID=true
//go:generate go run .. --bundle=true --input bundle/yaml.yaml --output bundle/yaml.schema.json
//go:generate go run .. --bundle=true --input bundle/simple.yaml --output bundle/simple-root-ref.schema.json --schemaRoot.ref ./bundle/simple-subschema.schema.json
