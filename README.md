# helm values schema json plugin

[![ci](https://github.com/losisin/helm-values-schema-json/actions/workflows/ci.yaml/badge.svg)](https://github.com/losisin/helm-values-schema-json/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/losisin/helm-values-schema-json/graph/badge.svg?token=0QQVCFJH84)](https://codecov.io/gh/losisin/helm-values-schema-json)
[![Static Badge](https://img.shields.io/badge/licence%20-%20MIT-green)](https://github.com/losisin/helm-values-schema-json/blob/add-Makefile/LICENSE)
[![GitHub release (with filter)](https://img.shields.io/github/v/release/losisin/helm-values-schema-json)](https://github.com/losisin/helm-values-schema-json/releases)


Helm plugin for generating `values.schema.json` from single or multiple values files. Works only with Helm3 charts.

## Install

```bash
$ helm plugin install https://github.com/losisin/helm-values-schema-json.git
Installed plugin: schema
```

## Features

- Add multiple values files and merge them together - required
- Save output with custom name and location - default is values.schema.json in current working directory
- Change schema draft version - default is draft 2020-12

## Usage

```bash
$ helm schema -help
usage: helm schema [-input STR] [-draft INT] [-output STR]
  -draft int
    	Draft version (4, 6, 7, 2019, or 2020) (default 2020)
  -input value
    	Multiple yamlFiles as inputs (comma-separated)
  -output string
    	Output file path (default "values.schema.json")
```

### Basic

In most cases you will want to run the plugin with default options:

```bash
$ helm schema -input values.yaml
```

This will read `values.yaml`, set draft version to `2020-12` and save outpout to `values.schema.json`.

### Extended

Merge multiple values files, set json-schema draft version explicitly and save output to `my.schema.json`:

`values_1.yaml`

```yaml
nodeSelector:
  kubernetes.io/hostname: ""
dummyList:
  - "a"
  - "b"
  - "c"
key1: "asd"
key2: 42
key3: {}
key4: []
```

`custom/path/values_2.yaml`

```yaml
nodeSelector:
  kubernetes.io/hostname: "node1"
deep:
  deep1:
    deep2:
      deep3:
        deep4: "asdf"
```

Run the following command to merge the yaml files and output json schema:

```bash
$ helm schema -input values_1.yaml,custom/path/values_2.yaml -draft 7 -output my.schema.json
```

Output will be something like this:

```json
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
        "deep": {
            "type": "object",
            "properties": {
                "deep1": {
                    "type": "object",
                    "properties": {
                        "deep2": {
                            "type": "object",
                            "properties": {
                                "deep3": {
                                    "type": "object",
                                    "properties": {
                                        "deep4": {
                                            "type": "string"
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        },
        "dummyList": {
            "type": "array",
            "items": {
                "type": "string"
            }
        },
        "key1": {
            "type": "string"
        },
        "key2": {
            "type": "integer"
        },
        "key3": {
            "type": "object"
        },
        "key4": {
            "type": "array"
        },
        "nodeSelector": {
            "type": "object",
            "properties": {
                "kubernetes.io/hostname": {
                    "type": "string"
                }
            }
        }
    }
}
```