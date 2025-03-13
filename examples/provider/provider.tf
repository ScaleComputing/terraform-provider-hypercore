provider "hypercore" {
  host        = "https://hypercore-host-url"
  username    = "hypercore-username"
  password    = "hypercore-password"
  auth_method = "local"
  timeout     = 60.0

  # These credentials are all optional and can also be set as environment variables
  # HC_HOST=https://hypercore-host-url
  # HC_USERNAME=hypercore-username
  # HC_PASSWORD=hypercore-password
  # HC_AUTH_METHOD=local
  # HC_TIMEOUT=60.0
}
