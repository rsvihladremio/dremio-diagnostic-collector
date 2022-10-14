#!/bin/sh

# script/release: build binaries in all supported platforms and upload them with the gh client

param(
     $VERSION
)

Set-Location "$PSScriptRoot\.."

# this is also set in script/build and is a copy paste
$GIT_SHA=@(git rev-parse --short HEAD)
$LDFLAGS="-X github.com/rsvihladremio/dremio-diagnostic-collector/cmd.GitSha=$GIT_SHA -X github.com/rsvihladremio/dremio-diagnostic-collector/cmd.Version=$VERSION"

Write-Output "Cleaning bin folder…"
Get-Date
./script/clean

Write-Output "Building linux-amd64…"
Get-Date
$Env:GOOS='linux' 
$Env:GOARCH='amd64'
go build -ldflags "$LDFLAGS" -o ./bin/ddc
zip ./bin/ddc-linux-amd64.zip ./bin/ddc

Write-Output "Building linux-arm64…"
Get-Date
$Env:GOOS='linux' 
$Env:GOARCH='arm64'
go build -ldflags "$LDFLAGS" -o ./bin/ddc
zip ./bin/ddc-linux-arm64.zip ./bin/ddc

Write-Output "Building darwin-os-x-amd64…"
Get-Date
$Env:GOOS='darwin' 
$Env:GOARCH='amd64'
go build -ldflags "$LDFLAGS" -o ./bin/ddc
zip ./bin/ddc-darwin-amd64.zip ./bin/ddc

Write-Output "Building darwin-os-x-arm64…"
Get-Date
$Env:GOOS='darwin' 
$Env:GOARCH='arm64'
go build -ldflags "$LDFLAGS" -o ./bin/ddc
zip ./bin/ddc-darwin-arm64.zip ./bin/ddc

Write-Output "Building windows-amd64…"
Get-Date
$Env:GOOS='windows' 
$Env:GOARCH='amd64'
go build -ldflags "$LDFLAGS" -o ./bin/ddc.exe
zip ./bin/ddc-windows-amd64.zip ./bin/ddc.exe

Write-Output "Building windows-arm64…"
Get-Date
$Env:GOOS='windows' 
$Env:GOARCH='arm64'
GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o ./bin/ddc.exe
zip ./bin/ddc-windows-amd64.zip ./bin/ddc.exe

gh release create $VERSION --title $VERSION -F changelog.md ./bin/ddc-windows-arm64.zip ./bin/ddc-windows-amd64.zip ./bin/ddc-darwin-arm64.zip ./bin/ddc-darwin-amd64.zip ./bin/ddc-linux-arm64.zip ./bin/ddc-linux-amd64.zip 
 
