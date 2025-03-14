# Annotations from comments

JSON schema is partially implemented in this tool. It uses line comments to add annotations for the schema because head comments are frequently used by humans and tools like helm-docs. Multiple annotations can be added to a single line separated by semicolon. For example:

```yaml
nameOverride: "myapp" # @schema maxLength:10;pattern:^[a-z]+$
```

This will generate following schema:

```json
"nameOverride": {
    "maxLength": 10,
    "type": "string"
}
```

The following annotations are supported:

* [Validation Keywords for Any Instance Type](#validation-keywords-for-any-instance-type)
    * [Type](#type)
    * [Multiple types](#multiple-types)
    * [Enum](#enum)
    * [ItemEnum](#itemEnum)
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
    * [unevaluatedProperties](#unevaluatedproperties)
* [Base URI, Anchors, and Dereferencing](#base-uri-anchors-and-dereferencing)
    * [$id](#id)
    * [$ref](#ref)
* [Meta-Data Annotations](#meta-data-annotations)
    * [title and description](#title-and-description)
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

Always returns array of strings. Special case is `null` where instead of string, it is treated as valid input type. [section 6.1.2](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.1.2)

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
            "80",
            "443",
            "8080",
            "8443"
        ]
    },
    "type": [
        "array"
    ]
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

```yaml
dummyList: # @schema uniqueItems:true
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

```yaml
image:
  repository: "nginx" # @schema required:true
  tag: "latest" # @schema required:true
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

JSON string added "AS IS" to the node. [section 10.3.2.2](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.3.2.2)

```yaml
image: # @schema patternProperties: {"^[a-z]$": {"type": "string"}}
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

Boolean. [section 10.3.2.3](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.3.2.3)

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

### unevaluatedProperties

Boolean. [section 10.3.2.4](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-11.3)

```yaml
image: # @schema unevaluatedProperties: false
  repository: "nginx"
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
}
```


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

### default

Any JSON value. [section 9.2](https://json-schema.org/draft/2020-12/json-schema-validation#section-9.2)

```yaml
tolerations: [] # @schema default: [{"key":"foo","operator":"Equal","value":"bar","effect":"NoSchedule"}]
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

```yaml
image:
  tag: latest # @schema readOnly: true
```

```json
"image": {
    "properties": {
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

Non-empty array. Each item of the array MUST be a valid JSON Schema. [section 10.2.1.1](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.2.1.1)

```yaml
image:
  digest: sha256:1234567890 # @schema allOf: [{"type": "string"}, {"minLength": 14}]
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

Non-empty array. Each item of the array MUST be a valid JSON Schema. [section 10.2.1.2](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.2.1.2)

```yaml
cluster:
  enabled: true
  nodes: 3 # @schema anyOf: [{"type": "number", "multipleOf": 3}, {"type": "number", "multipleOf": 5}]
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

Non-empty array. Each item of the array MUST be a valid JSON Schema. [section 10.2.1.3](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.2.1.3)

```yaml
MY_SECRET:
  ref: "secret-reference-in-manager"
  version: # @schema oneOf: [{"type": "string"}, {"type": "number"}]
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

A valid JSON Schema. [section 10.2.1.4](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-00#section-10.2.1.4)

```yaml
image:
  tag: latest # @schema not: {"type": "object"}
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
