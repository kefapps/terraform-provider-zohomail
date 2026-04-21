resource "zohomail_domain" "example" {
  domain_name = "example.com"
}

resource "zohomail_domain_catch_all" "example" {
  domain_name       = zohomail_domain.example.domain_name
  catch_all_address = "support@example.com"
}
