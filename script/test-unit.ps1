# script/test.ps1: Run test suite for application.

$ErrorActionPreference = "Stop"

Set-Location -Path (Get-Item (Split-Path -Parent $MyInvocation.MyCommand.Definition)).Parent.FullName


if ($env:DEBUG) {
    $DebugPreference = "Continue"
}

.\script\clean.ps1
.\script\build.ps1

$env:SKIP_INTEGRATION_SETUP = "true"
 go test -short -coverpkg=./... -coverprofile=covprofile ./...
