resource "local_file" "content_private_file" {
  filename = "${path.module}/dummy_private_key.pem"
  content  = file("${path.module}/dummy_private_key.pem")
}

resource "local_file" "content_certificate_file" {
  filename = "${path.module}/dummy_certificate.pem"
  content  = file("${path.module}/dummy_certificate.pem")
}

resource "azion_digital_certificate" "example1" {
  certificate_result = {
    name                = "New SSL certificate for www.terraformExample.com"
    certificate_content = local_file.content_certificate_file.content
    private_key         = local_file.content_private_file.content
  }
}

resource "azion_digital_certificate" "example2" {
  certificate_result = {
    name                = "New SSL certificate for www.terraformExample.com"
    certificate_content = file("${path.module}/dummy_certificate.pem")
    private_key         = file("${path.module}/dummy_private_key.pem")
  }
}

