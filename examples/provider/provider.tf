terraform {
  required_providers {
    zohomail = {
      source = "kefapps/zohomail"
    }
  }
}

provider "zohomail" {
  organization_id = var.zohomail_organization_id
  access_token    = var.zohomail_access_token
  data_center     = var.zohomail_data_center
}
