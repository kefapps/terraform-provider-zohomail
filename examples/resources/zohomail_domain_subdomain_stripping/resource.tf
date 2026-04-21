resource "zohomail_domain" "example" {
  domain_name = "example.com"
}

resource "zohomail_domain_subdomain_stripping" "example" {
  domain_name = zohomail_domain.example.domain_name
}
