provider "scale" {
  host        = "https://scale-host-url"
  username    = "scale-username"
  password    = "scale-password"
  auth_method = "local"
  timeout     = 60.0

  # These credentials are all optional and can also be set as environment variables
  # SC_HOST=https://scale-host-url
  # SC_USERNAME=scale-username
  # SC_PASSWORD=scale-password
  # SC_AUTH_METHOD=local
  # SC_TIMEOUT=60.0
}
