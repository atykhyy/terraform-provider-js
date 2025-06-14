---
page_title: "Provider: atykhyy/js"
description: |-
  The js provider is an experimental OpenTofu function provider which allows you to write Javascript helper functions.
  
---
# atykhyy/js Provider

This is an experimental OpenTofu function provider based on terraform-plugin-go and [Goja](https://github.com/dop251/goja), a pure Go ECMAScript 5 runtime.
It allows you to write Javascript helper functions next to your Tofu code, so that you can use them in your Tofu configuration.

In OpenTofu 1.7.0 and upwards you can configure the provider and pass it Javascript source code as text.
- Only global functions with names beginning with an upper-case letter are exported to Tofu.
- The Tofu-facing name of the function **will be lower-cased**.
- Both simple types, like strings, integers, floats, and booleans, and complex types, like maps, slices, nullable pointers, and structures, are supported, both as arguments and return values.
- Being Javascript, there is very little type safety, so be careful.
- Functions receive copies of any Tofu objects passed in as arguments. Modifications will not propagate back to Tofu.

## Example Usage

```terraform
// main.tf
// note that ECMAScript interpolation fragments in inline strings must be escaped
provider "js" {
  js = <<-EOT
    function Hello(s) {
      return `Hello, $${s} !`
    }
  EOT
}

output "test" {
  value = provider::js::hello("papaya")
}
```
Output excerpt:
```
Changes to Outputs:
  + test = "Hello, papaya!"
```

## Argument Reference

The following arguments are supported:

* `js` - (String, Required) Javascript source code for your helper functions.
* `strict` - (Boolean, Optional) If `true`, configures the Javascript runtime to run in strict mode. Defaults to `true`.
