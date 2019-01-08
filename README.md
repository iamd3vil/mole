# Mole

Mole is a tool for creating SSH Tunnels using a declarative config. It can create multiple SSH tunnels. 

Mole expects a `config.yml` in the same directory or `/etc/mole/` or `$HOME/.mole/` directories. Configuration can be written in either `yaml` or `json`, mole can handle both formats.

### Example Config

```yaml
tunnels:
  - ssh_address: 103.230.194.142:22
    local_address: 127.0.0.1:4446
    remote_address: 127.0.0.1:4445
    ssh_user: root
    ssh_auth_method: password
    ssh_password: super-secret
```

### Config fields

- `ssh_address`: Address of the remote server which facilitates tunneling
- `local_address`: Local Ip and Port for the tunnel
- `remote_address`: Remote IP and Port for the tunnel
- `ssh_user`: Remote SSH user
- `ssh_auth_method`: Can be either `key` or `password`. Incase of `password`, `ssh_password` is mandatory. Default is `key`
- `ssh_password`: Password for SSH