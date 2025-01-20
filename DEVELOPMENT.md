# Development

## Setup CI

We use Github actions CI

Setup N self-hosted runners
- VM for runner
  - 8 vCPU
  - 4 GB RAM
  - 100 GB disk 
- Follow https://github.com/justinc1/terraform-provider-scale/settings/actions/runners/new
  - Install as regular user
  - Use directory `/opt/actions-runner-0` etc.
  - Run it as service - `sudo ./svc.sh install; sudo ./svc.sh start`

Add needed variables/secrets to github project:
- variable CI_CONFIG_HC_IP205, content
  ```
  SC_HOST=https://10.5.11.205
  SC_USERNAME=admin
  SC_PASSWORD=todo
  ```

TEMP: create VM named `tf-src`.
