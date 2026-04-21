resource "zohomail_domain" "example" {
  domain_name = "example.com"
}

resource "zohomail_domain_onboarding" "example" {
  domain_name         = zohomail_domain.example.domain_name
  verification_method = "txt"
  enable_mail_hosting = true
  verify_spf          = true
  verify_mx           = true
  make_primary        = false
}
