## Manual Install

Download the [latest release binaries](https://github.com/rsvihladremio/dremio-diagnostic-collector/releases/latest):

1. unzip the binary
2. open a terminal
3. change to the directory where you unzip your binary
4. run the command `./ddc -h` if you get the help for the command you are good to go.

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
