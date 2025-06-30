# Annotations from comments

JSON schema is partially implemented in this tool.

It supports both comments directly above, directly below,
and on the same line to add annotations for the schema.
The comment must start with `# @schema`, which is used to avoid interference
with tools like [helm-docs](https://github.com/norwoodj/helm-docs).

Multiple annotations can be added to a single line separated by semicolon.

Example:

```yaml
# On the same line
fullnameOverride: "myapp" # @schema maxLength:10;pattern:^[a-z]+$

# On the line above
# @schema maxLength:10;pattern:^[a-z]+$
nameOverride: "myapp"

# On the line below (double-check that indentation matches)
resources:
  limits: {}
  requests: {}
# @schema additionalProperties:false
```

This will generate following schema:

```json
"fullnameOverride": {
    "type": "string",
    "pattern": "^[a-z]+$",
    "maxLength": 10
},
"nameOverride": {
    "type": "string",
    "pattern": "^[a-z]+$",
    "maxLength": 10
},
"resources": {
    "type": "object",
    "properties": {
        "limits": {
            "type": "object"
        },
        "requests": {
            "type": "object"
        }
    },
    "additionalProperties": false
}
```

> aside: Support for comments above and below the property was introduced
> in v2.0.0. If you're using a version before v2.0.0 then only comments at the
> end of the same line is supported.

The following annotations are supported:

* [Validation Keywords for Any Instance Type](#validation-keywords-for-any-instance-type)
    * [Type](#type)
    * [Multiple types](#multiple-types)
    * [Enum](#enum)
    * [ItemEnum](#itemEnum)
    * [Const](#const)
* [Strings](#strings)
    * [maxLength](#maxlength)
    * [minLength](#minlength)
    * [pattern](#pattern)
* [Numbers](#numbers)
    * [multipleOf](#multipleof)
    * [maximum](#maximum)
    * [minimum](#minimum)
* [Arrays](#arrays)
    * [item](#item)
    * [maxItems](#maxitems)
    * [minItems](#minitems)
    * [uniqueItems](#uniqueitems)
* [Objects](#objects)
    * [minProperties](#minproperties)
    * [maxProperties](#maxproperties)
    * [required](#required)
    * [patternProperties](#patternproperties)
    * [additionalProperties](#additionalproperties)
* [Unevaluated Locations](#unevaluated-locations)
    * [unevaluatedProperties](#unevaluatedproperties)
* [Base URI, Anchors, and Dereferencing](#base-uri-anchors-and-dereferencing)
    * [$id](#id)
    * [$ref](#ref)
    * [$k8s alias](#k8s-alias)
    * [bundling](#bundling)
* [Meta-Data Annotations](#meta-data-annotations)
    * [title and description](#title-and-description)
    * [helm-docs](#helm-docs)
    * [examples](#examples)
    * [default](#default)
    * [readOnly](#readonly)
* [Schema Composition](#schema-composition)
    * [allOf](#allof)
    * [anyOf](#anyof)
    * [oneOf](#oneof)
    * [not](#not)

## Validation Keywords for Any Instance Type

### Type

The `type` keyword is used to restrict a value to a specific primitive type. There are several possible values for `type`:

* `string`
* `number`
* `integer`
* `boolean`
* `object`
* `array`
* `null`

### Multiple types

Default behaviour returns always a string unless annotation is used. In that case, it returns array of strings. Useful for keys without any value declared for documentation purposes. [section 6.1.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.1.1)

```yaml
# -- (int) replica count
replicaCount: # @schema type:[integer, null]
```

```json
"replicaCount": {
    "type": [
        "integer",
        "null"
    ]
}
```

Another way to use this is to define type when using anchors and aliases in yaml. See discussion [#28](https://github.com/losisin/helm-values-schema-json/issues/28) for more details.

```yaml
app: &app
  settings:
    namespace:
      - *app # @schema type:[string]
```

```json
{
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "properties": {
        "app": {
            "properties": {
                "settings": {
                    "properties": {
                        "namespace": {
                            "items": {
                                "type": [
                                    "string"
                                ]
                            },
                            "type": "array"
                        }
                    },
                    "type": "object"
                }
            },
            "type": "object"
        }
    },
    "type": "object"
}
```

### Enum

Array of JSON values. [section 6.1.2](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.1.2)

```yaml
service: ClusterIP # @schema enum:[ClusterIP, LoadBalancer, null]
```

```json
"service": {
    "enum": [
        "ClusterIP",
        "LoadBalancer",
        null
    ],
    "type": "string"
}
```

### ItemEnum

This is a special key that apply [enum](#enum) on items of an array.

```yaml
port: [80, 443] # @schema itemEnum:[80, 8080, 443, 8443]
```

```json
"port": {
    "items": {
        "type": "number",
        "enum": [
            80,
            443,
            8080,
            8443
        ]
    },
    "type": [
        "array"
    ]
}
```

### Const

The `const` keyword is used to restrict instances to a single specific JSON value of any type including `null`. Therefore, `type` is redundant and dropped from generated schema. [section 6.1.3](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.1.3)

```yaml
nameOverride: foo # @schema const: foo
```

```json
"nameOverride": {
    "const": "foo"
}
```

## Strings

### maxLength

Non-negative integer. [section 6.3.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.3.1)

```yaml
nameOverride: "myapp" # @schema maxLength:10
```

This will generate following schema:

```json
"nameOverride": {
    "maxLength": 10,
    "type": "string"
}
```

### minLength

Non-negative integer. [section 6.3.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.3.2)

```yaml
nameOverride: "myapp" # @schema minLength:3
```

This will generate following schema:

```json
"nameOverride": {
    "minLength": 3,
    "type": "string"
}
```

### pattern

String that is valid regular expression, according to the ECMA-262 regular expression dialect. [section 6.3.3](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.3.3)

```yaml
nameOverride: "myapp" # @schema pattern:^[a-z]+$
```

This will generate following schema:

```json
"nameOverride": {
    "type": "string"
}
```

## Numbers

### multipleOf

Number greater than `0`. [section 6.2.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.2.1)

```yaml
replicas: 2 # @schema multipleOf:2
```

```json
"replicas": {
    "multipleOf": 2,
    "type": "integer"
}
```

### maximum

Number. [section 6.2.2](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.2.2)

```yaml
replicas: 2 # @schema maximum:10
```

```json
"replicas": {
    "maximum": 10,
    "type": "integer"
}
```

### minimum

Number. [section 6.2.4](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.2.4)

```yaml
replicas: 5 # @schema minimum:2
```

```json
"replicas": {
    "minimum": 2,
    "type": "integer"
}
```

## Arrays

### item

Define the item type of empty arrays.

```yaml
imagePullSecrets: [] # @schema item: object
```

This will generate following schema:

```json
"imagePullSecrets": {
    "items": {
        "type": "object"
    }
}
```

### maxItems

Non-negative integer. [section 6.4.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.4.1)

```yaml
dummyList: # @schema maxItems:5
  - "item1"
  - "item2"
  - "item3"
```

```json
"dummyList": {
    "items": {
        "type": "string"
    },
    "maxItems": 5,
    "type": "array"
}
```

### minItems

Non-negative integer. [section 6.4.2](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.4.2)

```yaml
dummyList: # @schema minItems:2
  - "item1"
  - "item2"
  - "item3"
```

```json
"dummyList": {
    "items": {
        "type": "string"
    },
    "minItems": 2,
    "type": "array"
}
```

### uniqueItems

Boolean. [section 6.4.3](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.4.3)

The `:true` part of `uniqueItems:true` is optional.

```yaml
dummyList: # @schema uniqueItems:true
  - "item1"
  - "item2"
  - "item3"

otherList: # @schema uniqueItems
  - "item1"
  - "item2"
  - "item3"
```

```json
"dummyList": {
    "items": {
        "type": "string"
    },
    "type": "array",
    "uniqueItems": true
},
"otherList": {
    "items": {
        "type": "string"
    },
    "type": "array",
    "uniqueItems": true
}
```

## Objects

### maxProperties

Non-negative integer. [section 6.5.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.5.1)

```yaml
nodeSelector: # @schema maxProperties:10
  kubernetes.io/hostname: "my-node"
```

```json
"nodeSelector": {
    "maxProperties": 10,
    "properties": {
        "kubernetes.io/hostname": {
            "type": "string"
        }
    },
    "type": "object"
}
```

### minProperties

Non-negative integer. [section 6.5.2](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.5.2)

```yaml
nodeSelector: # @schema minProperties:1
  kubernetes.io/hostname: "my-node"
```

```json
"nodeSelector": {
    "minProperties": 1,
    "properties": {
        "kubernetes.io/hostname": {
            "type": "string"
        }
    },
    "type": "object"
}
```

### required

Array of unique strings appended to the parent node. [section 6.5.3](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.5.3)

The `:true` part of `required:true` is optional.

```yaml
image:
  repository: "nginx" # @schema required:true
  tag: "latest" # @schema required
  imagePullPolicy: "IfNotPresent"
```

```json
"image": {
    "properties": {
        "imagePullPolicy": {
            "type": "string"
        },
        "repository": {
            "type": "string"
        },
        "tag": {
            "type": "string"
        }
    },
    "required": [
        "repository",
        "tag"
    ],
    "type": "object"
}
```

### patternProperties

YAML object added "AS IS" to the node. [section 10.3.2.2](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.3.2.2)

```yaml
image: # @schema patternProperties: {"^[a-z]$": {type: string}}
  repository: "nginx"
```

```json
"image": {
    "patternProperties": {
        "^[a-z]$": {
            "type": "string"
        }
    },
    "properties": {
        "repository": {
            "type": "string"
        }
    },
    "type": "object"
}
```

### additionalProperties

YAML object of a Schema or boolean. [section 10.3.2.3](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.3.2.3)

```yaml
image: # @schema additionalProperties: false
  repository: "nginx"
```

```json
"image": {
    "additionalProperties": false,
    "properties": {
        "repository": {
            "type": "string"
        }
    },
    "type": "object"
}
```

```yaml
image: {} # @schema additionalProperties: {type: string}
```

```json
"image": {
    "additionalProperties": {
        "type": "string"
    },
    "type": "object"
}
```

## Unevaluated Locations

### unevaluatedProperties

Boolean. [section 11.3](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-11.3)

The `:true` part of `unevaluatedProperties:true` is optional.

```yaml
image: # @schema unevaluatedProperties: false
  repository: "nginx"

secrets: # @schema unevaluatedProperties
  foo: "bar"
```

```json
"image": {
    "unevaluatedProperties": false,
    "properties": {
        "repository": {
            "type": "string"
        }
    },
    "type": "object"
},
"secrets": {
    "unevaluatedProperties": true,
    "properties": {
        "foo": {
            "type": "string"
        }
    },
    "type": "object"
}
```

## Base URI, Anchors, and Dereferencing

### $id

String. [section 8.2.1](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-8.2.1)

```yaml
image: # @schema $id: https://example.com/schema.json
  repository: nginx
  tag: latest
  pullPolicy: Always
```

```json
"image": {
    "$id": "https://example.com/schema.json",
    "properties": {
        "pullPolicy": {
            "type": "string"
        },
        "repository": {
            "type": "string"
        },
        "tag": {
            "type": "string"
        }
    },
    "type": "object"
}
```

### $ref

String. [section 8.2.3.1](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-8.2.3.1)

```yaml
subchart: # @schema $ref: https://example.com/schema.json
  enabled: true
  name: subchart
  values:
    foo: bar
    bar: baz
```

```json
"subchart": {
    "$ref": "https://example.com/schema.json",
    "properties": {
        "enabled": {
            "type": "boolean"
        },
        "name": {
            "type": "string"
        },
        "values": {
            "properties": {
                "bar": {
                    "type": "string"
                },
                "foo": {
                    "type": "string"
                }
            },
            "type": "object"
        }
    },
    "type": "object"
}
```

(since v1.9.0) When targeting JSON Schema Draft 7 or earlier
(e.g via `--draft 7` flag), then the resulting schema will use `allOf` like so:

```json
"subchart": {
    "allOf": [
        {
            "$ref": "https://example.com/schema.json",
        },
        {
            "properties": {
                "enabled": {
                    "type": "boolean"
                },
                "name": {
                    "type": "string"
                },
                "values": {
                    "properties": {
                        "bar": {
                            "type": "string"
                        },
                        "foo": {
                            "type": "string"
                        }
                    },
                    "type": "object"
                }
            },
            "type": "object"
        }
    ]
}
```

### $k8s alias

(since v1.9.0)

You can use `$ref: $k8s/...` as a shorthand for
`https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/{{ .K8sSchemaVersion }}/...`

To use that you must provide the `--k8s-schema-version` flag or `k8s-schema-version`
config. For example:

```bash
helm schema --values values.yaml --k8s-schema-version v1.33.1
```

```yaml
# values.yaml

memory: # @schema $ref: $k8s/_definitions.json#/definitions/io.k8s.apimachinery.pkg.api.resource.Quantity
```

```json
{
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "type": "object",
    "properties": {
        "memory": {
            "$ref": "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/v1.33.1/_definitions.json#/definitions/io.k8s.apimachinery.pkg.api.resource.Quantity",
            "type": "string"
        }
    }
}
```

### bundling

(Since v1.9.0)

You can bundle referenced schemas, which will resolve the `$ref` and embeds the
result in `$defs`. To enable bundling, use the command-line flags:

- `--bundle` enables bundling (default: `false`)

- `--bundle-root /some/path` sets the root directory from which file `$ref` are
  allowed to read files from (default: current working directory)

- `--bundle-without-id` works as a compatibility mode by disabling usage of
  `$id` and overriding `$ref` with syntax like `"$ref": "#/$defs/schema.json"`
  instead of retaining the original `$ref`. This is helpful for VSCode and
  other editors using Microsoft's JSON language server as that implementation
  [does not support the `$id` keyword](https://github.com/microsoft/vscode-json-languageservice/issues/224).

  Helm does support `$id`. So this setting is only for better editor
  integration.

Bundling supports the following schemes:

```yaml
## HTTP & HTTPS
# @schema $ref: http://example.com/schema.json
# @schema $ref: http://example.com/schema.yaml
# @schema $ref: https://example.com/schema.json
# @schema $ref: https://example.com/schema.yaml

## Local files
## NOTE: "file://" only supports absolute paths
# @schema $ref: file:///some/absolute/path.json
# @schema $ref: file:///some/absolute/path.yaml
# @schema $ref: /some/absolute/path.json
# @schema $ref: /some/absolute/path.yaml
# @schema $ref: some/relative/path.json
# @schema $ref: some/relative/path.yaml
# @schema $ref: ./some/relative/path.json
# @schema $ref: ./some/relative/path.yaml

## Local schema references are not bundled. They are kept as-is.
# @schema $ref: #/properties/foobar
```

## Meta-Data Annotations

### title and description

String. [section 9.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-9.1)

```yaml
fullnameOverride: bar # @schema title: My title ; description: My description
```

```json
"fullnameOverride": {
    "title": "My title",
    "description": "My description",
    "type": "string"
},
```

### helm-docs

(since v2.0.0)

Use description from <https://github.com/norwoodj/helm-docs> comments.
Must be enabled to be used via the `--use-helm-docs` flag.

```bash
helm schema --use-helm-docs
```

```yaml
# -- My description
fullnameOverride: bar
```

```json
"fullnameOverride": {
    "description": "My description",
    "type": "string"
},
```

The following helm-docs features are not supported:

- Helm-docs specific properties:

  - `# @default --`
  - `# @section --`
  - *etc.*

- Detached comments. Meaning, comments that are not directly above the property.
  For example:

  ```yaml
  # fullnameOverride -- This works
  fullnameOverride: bar
  ```

  ```yaml
  fullnameOverride: bar

  # fullnameOverride -- This does not work. Helm-docs will see the comment,
  # but this schema plugin will not.
  ```

While this plugin supports helm-docs, helm-docs does not support this plugin.
So on comments above the field must have the `# @schema` comments
above the helm-docs description comment to avoid having the schema annotations
getting included in the description:

```yaml
# ✅ good:
# @schema maxLength:10
# -- My awesome nameOverride description
nameOverride: "myapp"

# ❌ bad:
# -- My awesome nameOverride description
# @schema maxLength:10
nameOverride: "myapp"
```

### examples

(since v2.1.0)

Array of JSON values. [section 9.5](https://json-schema.org/draft/2020-12/json-schema-validation#section-9.5)

```yaml
tag: "" # @schema examples: [v1.2.3, v1.2.3-beta1]
```

```json
"tag": {
    "examples": [
        "v1.2.3",
        "v1.2.3-beta1"
    ],
    "type": "string"
}
```

### default

Any YAML value. [section 9.2](https://json-schema.org/draft/2020-12/json-schema-validation#section-9.2)

```yaml
tolerations: [] # @schema default: [{key: foo, operator: Equal, value: bar, effect: NoSchedule}]
```

```json
"tolerations": {
    "default": [
        {
            "effect": "NoSchedule",
            "key": "foo",
            "operator": "Equal",
            "value": "bar"
        }
    ],
    "type": "array"
}
```

### readOnly

Boolean. [section 9.4](https://json-schema.org/draft/2020-12/json-schema-validation#section-9.4)

The `:true` part of `readOnly:true` is optional.

```yaml
image:
  repository: "nginx" # @schema readOnly:true
  tag: "latest" # @schema readOnly
```

```json
"image": {
    "properties": {
        "repository": {
            "readOnly": true,
            "type": "string"
        },
        "tag": {
            "readOnly": true,
            "type": "string"
        }
    },
    "type": "object"
}
```

## Schema Composition

Keywords for Applying Subschemas With Logic. Field `"type"` is dropped and you MUST declare it as part of the schema provided for the keyword.

### allOf

Non-empty YAML array. Each item of the array MUST be a valid Schema. [section 10.2.1.1](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.2.1.1)

```yaml
image:
  digest: sha256:1234567890 # @schema allOf: [{type: string}, {minLength: 14}]
```

```json
"image": {
    "properties": {
        "digest": {
            "allOf": [
                {
                    "type": "string"
                },
                {
                    "minLength": 14
                }
            ]
        }
    },
    "type": "object"
}
```

### anyOf

Non-empty YAML array. Each item of the array MUST be a valid Schema. [section 10.2.1.2](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.2.1.2)

```yaml
cluster:
  enabled: true
  nodes: 3 # @schema anyOf: [{type: number, multipleOf: 3}, {type: number, multipleOf: 5}]
```

```json
"cluster": {
    "properties": {
        "enabled": {
            "type": "boolean"
        },
        "nodes": {
            "anyOf": [
                {
                    "multipleOf": 3,
                    "type": "number"
                },
                {
                    "multipleOf": 5,
                    "type": "number"
                }
            ]
        }
    },
    "type": "object"
}
```

### oneOf

Non-empty YAML array. Each item of the array MUST be a valid Schema. [section 10.2.1.3](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.2.1.3)

```yaml
MY_SECRET:
  ref: "secret-reference-in-manager"
  version: # @schema oneOf: [{type: string}, {type: number}]
```

```json
"MY_SECRET": {
    "properties": {
        "ref": {
            "type": "string"
        },
        "version": {
            "oneOf": [
                {
                    "type": "string"
                },
                {
                    "type": "number"
                }
            ]
        }
    },
    "type": "object"
}
```

### not

YAML object. MUST be a valid Schema. [section 10.2.1.4](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.2.1.4)

```yaml
image:
  tag: latest # @schema not: {type: object}
```

```json
"image": {
    "properties": {
        "tag": {
            "not": {
                "type": "object"
            }
        }
    },
    "type": "object"
}
```
