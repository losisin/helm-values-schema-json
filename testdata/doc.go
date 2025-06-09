// Package testdata contains files used by tests.
//
// To regenerate the test data output files, run the following:
//
//	go generate ./testdata
package testdata

//go:generate go run .. --values anchors.yaml --output anchors.schema.json
//go:generate go run .. --values basic.yaml --output basic.schema.json
//go:generate go run .. --values full.yaml --output full.schema.json --schema-root.id https://example.com/schema --schema-root.ref schema/product.json --schema-root.title "Helm Values Schema" --schema-root.description "Schema for Helm values" --schema-root.additional-properties=true
//go:generate go run .. --values k8sRef.yaml --output k8sRef.schema.json --k8s-schema-version v1.33.1
//go:generate go run .. --values meta.yaml --output meta.schema.json
//go:generate go run .. --values noAdditionalProperties.yaml --output noAdditionalProperties.schema.json --no-additional-properties=true
//go:generate go run .. --values ref.yaml --output ref-draft2020.schema.json --draft 2020
//go:generate go run .. --values ref.yaml --output ref-draft7.schema.json --draft 7
//go:generate go run .. --values subschema.yaml --output subschema.schema.json

//go:generate go run .. --use-helm-docs --values helm-docs/values.yaml --output helm-docs/values.schema.json

//go:generate go run .. --bundle=false --values bundle/simple.yaml --output bundle/simple-disabled.schema.json
//go:generate go run .. --bundle=true --values bundle/fragment.yaml --output bundle/fragment-without-id.schema.json --bundle-without-id=true
//go:generate go run .. --bundle=true --values bundle/fragment.yaml --output bundle/fragment.schema.json
//go:generate go run .. --bundle=true --values bundle/multiple-values-1.yaml,bundle/multiple-values-2.yaml --schema-root.ref bundle/simple-subschema.schema.json --output bundle/multiple-values-without-id.schema.json --bundle-without-id=true
//go:generate go run .. --bundle=true --values bundle/multiple-values-1.yaml,bundle/multiple-values-2.yaml --schema-root.ref bundle/simple-subschema.schema.json --output bundle/multiple-values.schema.json
//go:generate go run .. --bundle=true --values bundle/namecollision.yaml --output bundle/namecollision.schema.json
//go:generate go run .. --bundle=true --values bundle/nested.yaml --output bundle/nested-without-id.schema.json --bundle-without-id=true
//go:generate go run .. --bundle=true --values bundle/nested.yaml --output bundle/nested.schema.json
//go:generate go run .. --bundle=true --values bundle/simple.yaml --output bundle/simple-absolute-root.schema.json --bundle-root=/
//go:generate go run .. --bundle=true --values bundle/simple.yaml --output bundle/simple-root-ref.schema.json --schema-root.ref ./bundle/simple-subschema.schema.json
//go:generate go run .. --bundle=true --values bundle/simple.yaml --output bundle/simple-without-id.schema.json --bundle-without-id=true
//go:generate go run .. --bundle=true --values bundle/simple.yaml --output bundle/simple.schema.json
//go:generate go run .. --bundle=true --values bundle/yaml.yaml --output bundle/yaml.schema.json
