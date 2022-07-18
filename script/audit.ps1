# script/audit: runs gosec against the mod file to find security issues
#                   

Set-Location "$PSScriptRoot\.."

gosec ./...
