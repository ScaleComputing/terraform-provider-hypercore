# Development

## Setup CI

We use Github actions CI

Setup N self-hosted runners
- VM for runner
  - 8 vCPU
  - 4 GB RAM
  - 100 GB disk 
- Follow https://github.com/ScaleComputing/terraform-provider-hypercore/settings/actions/runners/new
  - Install as regular user
  - Use directory `/opt/actions-runner-0` etc.
  - Run it as service - `sudo ./svc.sh install; sudo ./svc.sh start`

Add needed variables/secrets to github project:
- variable CI_CONFIG_HC_IP205, content
  ```
  HC_HOST=https://10.5.11.205
  HC_USERNAME=admin
  HC_PASSWORD=todo
  ```

TEMP: create VM named `testtf-src`.

Prior to running acceptance tests we need to setup:
  1. Virtual machine
  2. Virtual disk prior
  3. Add names and UUIDs to the env.txt file in /tests/acceptance/setup directory
  4. Virtual machine needs to be powered off
