## Manual Install

Download the latest release v0.1.4 binaries:

* [ddc-darwin-amd64.zip](https://github.com/rsvihladremio/dremio-diagnostic-collector/releases/download/v0.1.4/ddc-darwin-amd64.zip)
* [ddc-darwin-arm64.zip](https://github.com/rsvihladremio/dremio-diagnostic-collector/releases/download/v0.1.4/ddc-darwin-arm64.zip)
* [ddc-linux-amd64.zip](https://github.com/rsvihladremio/dremio-diagnostic-collector/releases/download/v0.1.4/ddc-linux-amd64.zip)
* [ddc-linux-arm64.zip](https://github.com/rsvihladremio/dremio-diagnostic-collector/releases/download/v0.1.4/ddc-linux-arm64.zip)
* [ddc-windows-amd64.zip](https://github.com/rsvihladremio/dremio-diagnostic-collector/releases/download/v0.1.4/ddc-windows-amd64.zip)

1. unzip the binary
2. open a terminal
3. change to the directory where you unzip your binary
4. run the command `./ddc-h` if you get the help for the command you are good to go.

## Automatic Installation

### Mac

Use hombrew

```sh
brew tap rsvihladremio/ddc
brew install ddc
```

### Linux

Use shell script

```sh
/bin/bash -c "$(curl https://raw.githubusercontent.com/rsvihladremio/dremio-diagnostic-collector/main/script/install)"
```

### Windows

Use powershell script

```pwsh
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser # Optional: Needed to run a remote script the first time
irm https://raw.githubusercontent.com/rsvihladremio/dremio-diagnostic-collector/main/script/install.ps1  | iex 
```
