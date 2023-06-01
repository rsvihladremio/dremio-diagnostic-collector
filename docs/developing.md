## Developing

### Dependency

1. Recent verision of Docker up and running
2. Access to the dremio-ee Docker image
3. Log into docker using the [docker login](https://docs.docker.com/engine/reference/commandline/login/) command
4. Pull down dremio-ee with the following command before running any tests `docker pull dremio/dremio-ee:24.0`

### Scripts

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
```


