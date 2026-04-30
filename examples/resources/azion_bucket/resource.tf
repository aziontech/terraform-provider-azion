resource "azion_bucket" "example" {
  bucket = {
    name              = "my-bucket-name"
    workloads_access  = "read_write"
  }
}
