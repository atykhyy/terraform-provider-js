# terraform-provider-js

This is an experimental OpenTofu function provider based on terraform-plugin-go and Goja, a pure Go ECMAScript runtime.

It allows you to write ECMAScript helper functions next to your Tofu code, so that you can use them in your Tofu configuration. The provider is based on [Goja](https://github.com/dop251/goja).

In OpenTofu 1.7.0 and upwards you can configure the provider and pass it a Javascript file to load (or use here-text).
- Exported functions need to start with upper-case letters.
- The Tofu-facing name of the function **will be lower-cased**.
- It supports simple types, like strings, integers, floats, and booleans.
- It also supports complex type, like maps, slices, nullable pointers, and structures.
- Being ECMAScript, there is very little type safety, so be careful.

## Example

```hcl
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

## Importing
Here's a snippet to require the provider in your OpenTofu configuration:
```hcl
terraform {
  required_providers {
    go = {
      source  = "registry.opentofu.org/atykhyy/js"
      version = "0.0.1"
    }
  }
}
```
