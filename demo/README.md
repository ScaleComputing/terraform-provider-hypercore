Prepare:
 - Create VM named `testtf-src-empty`, with 0 disks and 0 nics.
 - Since demo does not try to import existing resources:
    - Remove VM `testtf-demo`, if it exists.
    - Remove virtual disk `jammy-server-cloudimg-amd64.img`, if it exists.
    - Remove ISO `alpine-virt-3.21.3-x86_64.iso`, if it exists

Follow [top level README.md](../README.md) to install terraform and golang.

Next install provider locally. Run:

```bash
# install provider locally
make install local_provider force_reinit_local
cd demo

export TF_LOG=info
export TF_CLI_ARGS_apply="-parallelism=1"
export HC_HOST=https://TODO_YOUR_HYPERCORE_HOST
export HC_USERNAME=admin
export HC_PASSWORD=TODO
export HC_TIMEOUT=600.0  # Virtual disk upload can be slow

terraform init
terraform validate

terraform plan
terraform apply -target hypercore_virtual_disk.ubuntu_2204
terraform apply

# shutdown VM before destroy
terraform destroy -target hypercore_vm.demo_vm
terraform destroy
```
