## Developing

On Linux, Mac, and WSL there are some shell scripts modeled off the [GitHub ones](https://github.com/github/scripts-to-rule-them-all)

to get started run

```sh
./script/bootstrap
```

after a pull it is a good idea to run

```sh
./script/update
```

tests

```sh
./script/test
```

before checkin run

```sh
./script/cibuild
```

to cut a release do the following

```sh
#dont forget to update changelog.md with the release notes
git tag v0.1.1
git push origin v0.1.1
./script/release v0.1.1
gh repo view -w
# review the draft and when done set it to publish
```
### Windows
Similarly on Windows there are powershell scripts of the same design

to get started run

```powershell
.\script\bootstrap.ps1
```

after a pull it is a good idea to run

```powershell
.\script\update.ps1
```

tests

```powershell
.\script\test.ps1
```

before checkin run

```powershell
.\script\cibuild.ps1
```

to cut a release do the following

```powershell
#dont forget to update changelog.md with the release notes
git tag v0.1.1
.\script\release.ps1 v0.1.1
gh repo view -w
# review the draft and when done set it to publish
```

