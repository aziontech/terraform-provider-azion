resource "azion_edge_application_edge_functions_instance" "example" {
    edge_application_id = <edge_application_id>
    results = {
    name = "Terraform Example"
    "edge_function_id": <edge_function_id>,
    "args": jsonencode(
            { "key" = "Value",
              "Example" = "example"
            })
    }
}