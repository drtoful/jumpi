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

## License

JumPi is licensed under the BSD License. See LICENSE for more information.

