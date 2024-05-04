terraform {
  required_providers {
    js = {
      source  = "terraform.local/local/js"
      version = "0.0.1"
    }
  }
}

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
