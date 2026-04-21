resource "zohomail_domain" "example" {
  domain_name = "example.com"
}

resource "zohomail_domain_dkim" "example" {
  domain_name       = zohomail_domain.example.domain_name
  selector          = "terraform"
  hash_type         = "sha256"
  make_default      = true
  verify_public_key = false
}
