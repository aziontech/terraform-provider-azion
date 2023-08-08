resource "azion_edge_application_edge_functions_instance" "example" {
    edge_application_id = 1234567890
    results = {
    name = "Terraform Example"
    "edge_function_id": 12345,
    "args": jsonencode(
            { "key" = "Value",
              "Example" = "example"
            })
    }
}