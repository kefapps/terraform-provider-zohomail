resource "zohomail_mailbox" "support" {
  primary_email_address = "support@example.com"
  initial_password      = "replace-me"
  first_name            = "Support"
  last_name             = "Team"
  display_name          = "Support"
  role                  = "member"
  country               = "FR"
  language              = "fr"
  time_zone             = "Europe/Paris"
}

resource "zohomail_mailbox_forwarding" "support" {
  mailbox_id = zohomail_mailbox.support.id

  target_addresses = [
    "sales@example.com",
    "hello@example.com",
  ]

  delete_zoho_mail_copy = false
}
