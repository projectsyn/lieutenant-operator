path "kv/data/*" {
  capabilities = ["read", "create", "update", "delete"]
}

path "kv/metadata/*" {
  capabilities = ["read", "create", "update", "delete", "list"]
}

path "kv/delete/*" {
  capabilities = ["update"]
}
