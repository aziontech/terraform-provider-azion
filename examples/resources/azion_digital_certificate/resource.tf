resource "local_file" "content_private_file" {
  filename = "${path.module}/private_key.pem"
  content  = file("${path.module}/private_key.pem")
}

resource "local_file" "content_certificate_file" {
  filename = "${path.module}/certificate.pem"
  content  = file("${path.module}/certificate.pem")
}

resource "azion_digital_certificate" "example" {
  certificate_result = {
    name  = "New SSL certificate for www.terraformExample.com"
    certificate_content = local_file.content_certificate_file.content
    private_key = local_file.content_private_file.content
  }
}

resource "azion_digital_certificate" "example" {
  certificate_result = {
    name  = "New SSL certificate for www.terraformExample.com"
    certificate_content = file("${path.module}/certificate.pem")
    private_key = file("${path.module}/private_key.pem")
  }
}

