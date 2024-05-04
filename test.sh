go build

dest=~/.terraform.d/plugins/terraform.local/local/js/0.0.1/darwin_arm64/terraform-provider-js_v0.0.1
mkdir -p $(dirname $dest)

cp terraform-provider-js $dest

rm .terraform* -r
tofu init -reconfigure
tofu plan
