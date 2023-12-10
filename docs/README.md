# Annotations from comments

JSON schema is partially implemented in this tool. It uses line comments to add annotations for the schema because head comments are frequently used by humans and tools like helm-docs. The following annotations are supported:

* [Validation Keywords for Any Instance Type](#validation-keywords-for-any-instance-type)
    * [Type](#type)
    * [Enum](#enum)
* [Strings](#strings)
    * [maxLength](#maxlength)
    * [minLength](#minlength)
    * [pattern](#pattern)
* [Numbers](#numbers)
    * [minimum](#minimum)
    * [maximum](#maximum)
    * [multipleOf](#multipleof)
* [Booleans](#booleans)
* [Arrays](#arrays)
    * [minItems](#minitems)
    * [maxItems](#maxitems)
    * [uniqueItems](#uniqueitems)
    * [Items](#items)
* [Objects](#objects)
    * [minProperties](#minproperties)
    * [maxProperties](#maxproperties)
    * [required](#required)
* [Nulls](#nulls)

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

Default behaviour returns always a string unless annotation is used. In that case, it returns array of strings. [section 6.1.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.1.1)

### Enum

Always returns array of strings. Special case is `null` where instead of string, it is treated as valid inpput type.  [section 6.1.2](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.1.2)

## Strings

### maxLength

NOn-negative integer. [section 6.3.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.3.1)

### minLength

NOn-negative integer. [section 6.3.1](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.3.2)

### pattern

String that is valid regular expression, according to the ECMA-262 regular expression dialect. [section 6.3.3](https://json-schema.org/draft/2020-12/json-schema-validation#section-6.3.3)

