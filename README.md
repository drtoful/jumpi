# JumPi

## What is JumPi

JumPi is a priviledge access managment (PAM) system, that allows you to control and record activities
through SSH. Simplified, JumPi will act as a jumpstation that connects to a remote server via SSH and
records all keystrokes you do.

JumPi was created to run on a Raspberry Pi, a low-cost computer, but it can be run on any other 
computer or server as well.

### What happened to the old JumPi

Previously JumPi was made up of a bunch of python scripts. This worked, but had the impact, that connecting
to JumPi was really slow (as a whole python-env had to be forked). JumPi was thus rewritten from scratch
to act a SSH server itself to avoid forking binaries and slowing down. The old source-code is still
available under the 'python' tag in git.

## Installation

JumPi needs go 1.8 to compile. All dependencies are available within the source, using godeps. So you
can compile JumPi by using the following simple command.

    go get github.com/drtoful/jumpi

## Usage

After you have compiled JumPi you can start it, by issuing the following command

    jumpi

If this is the first time, you will be asked to provide a password for the admin account, as well
as a password for the store. The second password is used to unlock the store and allow JumPi to store
and read encrypted passwords into the DB.

JumPi has an API that can be used to configure targets and users to connect to. You can either use
your own UI or use the provided WSGI python application under 'www'.

You can see all command line options by using 

    jumpi -h

### MLock

Under linux you can use the '-mlock' option, which will mark all memory from JumPi as unswappable.
This is used as a security measure but uses a lot of memory, which you might not have on a Pi. Note
that all recordings are also stored in memory.

### Configuration

Before you can connect to any host, you will need to add a 'secret' to the database. 'secrets' are
either passwords or SSH RSA Keys, that can be used to connect to a server. Next you will need to
define a 'target'. This is a remote host with an associated 'secret' to it.

As the third step you will need to define a 'user'. A 'user' is defined by its SSH RSA Public Key
string. Lastly you will need to define a 'role' that defines, which user(s) can access which 
target(s).

#### User Configuration

Users can access individual configuration for their users under a special target (see **Connect**)
`config:<configuration>`. Note, that this target is subject to the defined roles. So you will need
to allow users to access configuration endpoints.

### Connect

After you have defined at least one secret, target, user and role you can connect to any of your
targets by using the following command:

    ssh -p2022 -o"User=<target>" jumpi

You will need to use your SSH RSA key associated to one of the users you defined previously (this
is how JumPi recognizes you and checks the role(s) you are in). If you use a different port than 
2022 you have to provide that.

Note that for 'target' you need to provide the full string as it appears in your UI. This means you
need to provide a string that has the following format:

    <user>@<host>:<port>

## Two-Factor Authentication

JumPi supports two factor authentication. The user can activate two-factor authentication for
their account by accessing the target `config:2fa:<type>`. Make sure you have a role that allows
users to access this special target.

Currently the following two-factor types are supported:

- [yubikey](https://www.yubico.com/products/yubikey-hardware/)

In addition you can specify roles that enforce two-factor authentication to access the
specified targets.

### Yubico Yubikey

In order to enable yubikey two-factor authentication you will need to store the API
key into the store under the name `config:yubikey_api`. The key must be stored as
**password** and in the form `<client_id>:<secret_key>`. You will see that it worked
in the output of JumPi.

Note, that currently only the official API servers are supported.

JumPi will honor `http_proxy` and `https_proxy` environment settings when connecting
to the API servers.

## Security

The source-code is currently not peer-reviewed for security.

Because go has its own memory managment, I have somewhat limited access and control what go does
with variables, and when and how it cleans its memory pages associated to it. Which means, that
passwords and secrets can linger in memory for quite some time before go is cleaning them up (if
at all).

Passwords and Secrets that get stored onto disk should be stored secure (in a somewhat similar
fashion as KeePass does) using ChaCha20 as a stream-cipher. Every password is encrypted using
a randomly generated password (which can in turn only decrypted via the store password), so if
you loose your store password, all passwords will be lost. So it is recommended that you store
them somewhere else (best in your own personal password safe).

## License

JumPi is licensed under the BSD License. See LICENSE for more information.

