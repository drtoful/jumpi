# JumPi

## What is JumPi

JumPi is a small collection of tools to create a SSH jumphost. As the
name suggests, it was developed to be used on a Raspberry Pi. But it
should run on any Linux system.

## Installing

    python setup.py install

### Dependencies

The following dependencies are automatically resolved by using the 
above command

* [flask](http://flask.pocoo.org/) >= 0.9
* [sqlalchemy](http://www.sqlalchemy.org/) >= 0.9
* [requests](http://docs.python-requests.org/en/latest/) >= 2.2.1
* [paramiko](http://www.paramiko.org/) >= 1.14
* [pyvault](https://github.com/drtoful/pyvault) >= 0.2
* [pyte](http://pyte.readthedocs.org/en/latest/) >= 0.4.8
* [alembic](http://alembic.readthedocs.org/en/latest/) >= 0.6.7

To successfully build and install the dependencies you might install
additional packages using your distributions package manager. For the
Raspberry Pi you can resolve this, by issuing the following command
(when using a raspbian based distribution:

    aptitude install build-essential python-dev libffi-dev libssl-dev

## Setting up

First you have to create a user, that will run all daemon and store information
about targets and users. This is usually called `jumpi`, but any name will work.
The user will be configured, so that it takes no password, but ssh-ing to it
will only work with private keys.

It's discouraged to use an existing user.

    adduser --system --shell /bin/sh --gecos 'ssh jumphost' \
    --group --disabled-password --home /home/jumpi jumpi

After this, you have to initialize the DB. For this login to your newly created
user (by using 'su') and issue the following command:

    jumpidb-create

If you are upgrading from a previous version you can issue the following
command instead to get a compatible DB:

    jumpidb-upgrade

Both the Web-UI and the agent are fully compliant WSGI applications. For the agent
use `from jumpi.agent import create_app`. For the Web-UI use `from jumpi.web import create_app`.
create\_app is a function, that returns a WSGI application.

If you choose non-standard ports to bind the applications to, you have to tell
them, on which port the agent can be accessed. Do this, by creating a file named
`jumpi-agent.cfg` in the home directory of the previously created user. In it
you can specify the agent host and port:

    [agent]
    host = 127.0.0.1
    port = 42000

There's a set of sample configuration files for uwsgi and nginx in the 'conf'
folder. We suggest using the emperor mode of uwsgi to handle loading of the
uwsgi processes.

We always suggest to bind the agent only on localhost (127.0.0.1) for security
reasons.

## Configuration

Once you login to the Web-UI, configuration should be straight forward. Before
you edit anything though, you have to unlock the agent. This you can do under
"System".

The default username and password for the Web-UI are both 'admin'. You can 
also do this under "System".

### Encryption

Encryption on the Raspberry Pi is not really what you could call efficient
or fast. Most of the time is consumed when deriving the "session" keys using
PBKDF2. The main time consumer is the number of iterations (naturally). You
can tune this in the file `jumpi-agent.cfg` in the home directory.

    [cipher]
    iterations = 500

This will change the number of iterations to 500 (half of the default 
iterations of 1000).

## License

JumPi is licensed under the BSD License. See LICENSE for more information.

