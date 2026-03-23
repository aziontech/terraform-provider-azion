resource "azion_workload" "example" {
  workload = {
    name           = "My Workload"
    active         = true
    infrastructure = 1

    tls = {
      certificate     = 1234
      ciphers         = 3
      minimum_version = "tls_1_2"
    }

    protocols = {
      http = {
        versions    = ["http1", "http2", "http3"]
        http_ports  = [80]
        https_ports = [443]
        quic_ports  = [443]
      }
    }

    mtls = {
      enabled = true
      config = {
        certificate  = 5678
        crl          = [9012]
        verification = "enforce"
      }
    }

    domains                      = ["example.com", "www.example.com"]
    workload_domain_allow_access = true
  }
}
