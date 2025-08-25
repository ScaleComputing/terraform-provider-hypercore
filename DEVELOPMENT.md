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
  TF_LOG=info
  TF_CLI_ARGS_apply="-parallelism=1" # Platform constraints (does not support parallelism)
  HC_TIMEOUT=600.0
  HC_HOST=https://10.5.11.205
  HC_USERNAME=admin
  HC_PASSWORD=todo
  SMB_SERVER=10.5.11.36
  SMB_USERNAME=";administrator"
  SMB_PASSWORD="todo"
  SMB_PATH=/cidata
  SMB_FILENAME=bla.xml
  ```

Prior to running acceptance tests we need to setup:
  - Virtual machine
    - name integration-test-vm
    - has two disks
      - type VIRTIO (1.2GB, 2.4GB)
    - has two nics
      - first of type INTEL_E1000, vlan 10, MAC 7C:4C:58:12:34:56
      - second of type VIRTIO, vlan ALL
    - boot order is configured as [disk, nic]
  - Virtual disk (as standalone not attached to the testing VM)
  - Add names and UUIDs to the env.txt file in /tests/acceptance/setup directory
    - There are multiple env-*.txt files, for different test HyperCore clusters.
  - Virtual machine needs to be powered off
