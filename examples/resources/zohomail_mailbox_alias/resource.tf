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

resource "zohomail_mailbox_alias" "sales" {
  mailbox_id  = zohomail_mailbox.support.id
  email_alias = "sales@example.com"
}
