# helm values schema json plugin

[![ci](https://github.com/losisin/helm-values-schema-json/actions/workflows/ci.yaml/badge.svg)](https://github.com/losisin/helm-values-schema-json/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/losisin/helm-values-schema-json/graph/badge.svg?token=0QQVCFJH84)](https://codecov.io/gh/losisin/helm-values-schema-json)
[![Go Report Card](https://goreportcard.com/badge/github.com/losisin/helm-values-schema-json)](https://goreportcard.com/report/github.com/losisin/helm-values-schema-json)
[![Static Badge](https://img.shields.io/badge/licence%20-%20MIT-green)](https://github.com/losisin/helm-values-schema-json/blob/main/LICENSE)
[![GitHub release (with filter)](https://img.shields.io/github/v/release/losisin/helm-values-schema-json)](https://github.com/losisin/helm-values-schema-json/releases)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/losisin/helm-values-schema-json/total)


Helm plugin for generating `values.schema.json` from single or multiple values files. Schema can be enriched by reading annotations from comments. Works only with Helm3 charts.

## Installation

```bash
$ helm plugin install https://github.com/losisin/helm-values-schema-json.git
Installed plugin: schema
```

## Features

- Add multiple values files and merge them together - required
- Save output with custom name and location - default is values.schema.json in current working directory
- Use preferred schema draft version - default is draft 2020
- Read annotations from comments. See [docs](https://github.com/losisin/helm-values-schema-json/tree/main/docs) for more info or checkout example yaml files in [testdata](https://github.com/losisin/helm-values-schema-json/tree/main/testdata).

## Integrations

There are several ways to automate schema generation with this plugin. Main reason is that the json schema file can be hard to follow and we as humans tend to forget and update routine tasks. So why not automate it?

### GitHub actions

There is GitHub action that I've build using typescript and published on marketplace. You can find it [here](https://github.com/marketplace/actions/helm-values-schema-json-action). Basic usage is as follows:

```yaml
name: Generate values schema json
on:
  - pull_request
jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
      with:
        ref: ${{ github.event.pull_request.head.ref }}
      - name: Generate values schema json
        uses: losisin/helm-values-schema-json-action@v1
        with:
          input: values.yaml
```

### pre-commit hook

With pre-commit, you can ensure your JSON schema is kept up-to-date each time you make a commit.

First [install pre-commit](https://pre-commit.com/#install) and then create or update a `.pre-commit-config.yaml` in the root of your Git repo with at least the following content:

```yaml
repos:
  - repo: https://github.com/losisin/helm-values-schema-json
    rev: v1.5.2
    hooks:
      - id: helm-schema
        args: ["-input", "values.yaml"]
```

Then run:

```bash
pre-commit install
pre-commit install-hooks
```

Further changes to your chart files will cause an update to json schema when you make a commit.

### Husky

This is a great tool for adding git hooks to your project. You can find it's documentation [here](https://typicode.github.io/husky/). Here is how you can use it:

```json
"husky": {
  "hooks": {
    "pre-commit": "helm schema -input values.yaml"
  }
},
```

## Usage

```bash
$ helm schema -help
Usage: helm schema [options...] <arguments>
  -draft int
    	Draft version (4, 6, 7, 2019, or 2020) (default 2020)
  -indent int
    	Indentation spaces (even number) (default 4)
  -input value
    	Multiple yaml files as inputs (comma-separated)
  -output string
    	Output file path (default "values.schema.json")
  -schemaRoot.additionalProperties value
    	JSON schema additional properties (true/false)
  -schemaRoot.description string
    	JSON schema description
  -schemaRoot.id string
    	JSON schema ID
  -schemaRoot.title string
    	JSON schema title
```

### Configuration file

This plugin will look for it's configuration file called `.schema.yaml` in the current working directory. All options available from CLI can be set in this file. Example:

```yaml
# Required
input:
  - values.yaml

draft: 2020
indent: 4
output: values.schema.json

schemaRoot:
  id: https://example.com/schema
  title: Helm Values Schema
  description: Schema for Helm values
  additionalProperties: true
```

Then, just run the plugin without any arguments:

```bash
$ helm schema
```

### CLI

#### Basic

In most cases you will want to run the plugin with default options:

```bash
$ helm schema -input values.yaml
```

This will read `values.yaml`, set draft version to `2020-12` and save outpout to `values.schema.json`.

#### Extended

##### Multiple values files

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

##### Root JSON object properties

Adding ID, title and description to the schema:

`basic.yaml`

```yaml
image:
  repository: nginx
  tag: latest
  pullPolicy: Always
```

```bash
$ helm schema -input basic.yaml -schemaRoot.id "https://example.com/schema" -schemaRoot.title "My schema" -schemaRoot.description "This is my schema"
```

Generated schema will be:

```json
{
  "$id": "https://example.com/schema",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "description": "This is my schema",
  "properties": {
      "image": {
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
  },
  "title": "My schema",
  "type": "object"
}
```

## Issues, Features, Feedback

Your input matters. Feel free to open [issues](https://github.com/losisin/helm-values-schema-json/issues) for bugs, feature requests, or any feedback you may have. Check if a similar issue exists before creating a new one, and please use clear titles and explanations to help understand your point better. Your thoughts help me improve this project!

### How to Contribute

🌟 Thank you for considering contributing to my project! Your efforts are incredibly valuable. To get started:
1. Fork the repository.
2. Create your feature branch: `git checkout -b feature/YourFeature`
3. Commit your changes: `git commit -am 'Add: YourFeature'`
4. Push to the branch: `git push origin feature/YourFeature`
5. Submit a pull request! 🚀
