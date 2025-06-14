terraform {
  required_providers {
    js = {
      source  = "terraform.local/local/js"
      version = "0.0.1"
    }
  }
}

provider "js" {
  js = <<-js
    function Hello(s,x) {
      return {"a": `Hello, $${x} $${s[0]}!`, "b": [{"c":"1","d":3},{"d":[]},{"r":null},null]}
    }
  js
}

output "test" {
  value = provider::js::hello([null,"turd"], "a")
}
