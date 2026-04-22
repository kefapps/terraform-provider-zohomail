resource "zohomail_mailbox" "support" {
  primary_email_address = "support@example.com"
  initial_password      = var.mailbox_initial_password
  first_name            = "Support"
  last_name             = "Team"
  display_name          = "Support"
  role                  = "member"
  country               = "FR"
  language              = "fr"
  time_zone             = "Europe/Paris"
  one_time_password     = true
}
