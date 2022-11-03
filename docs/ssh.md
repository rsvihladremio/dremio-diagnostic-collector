
# Troubleshooting

## Not trusted key

If you get permission denied errors it is likely your ssh key is not accepted by the ssh servers, as the ssh based collection relies on [ssh public authentication](https://www.ssh.com/academy/ssh/public-key-authentication).
This is well documented in [Windows](https://docs.microsoft.com/en-us/windows-server/administration/openssh/openssh_keymanagement),
[Linux](https://www.redhat.com/sysadmin/key-based-authentication-ssh), and [Mac](https://www.linode.com/docs/guides/connect-to-server-over-ssh-on-mac/).
A quick way to get this working though on Mac, Linux and WSL is to use ssh-copy-id

## Incorrect or no ssh-user

if no ssh user is specified the default is empty and the command will not work without a specified user
