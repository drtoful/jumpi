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
* [python-daemon](https://pypi.python.org/pypi/python-daemon/) >= 1.6
* [sqlalchemy](http://www.sqlalchemy.org/) >= 0.9
* [requests](http://docs.python-requests.org/en/latest/) >= 2.2.1
* [paramiko](http://www.paramiko.org/) >= 1.14
* [pyvault](https://github.com/drtoful/pyvault) >= 0.1

To successfully build and install the dependencies you might install
additional packages using your distributions package manager. For the
Raspberry Pi you can resolve this, by issuing the following command
(when using a raspbian based distribution:

    aptitude install build-essential python-dev libffi-dev

## Setting up

First you have to create a user, that will run all daemon and store information
about targets and users. This is usually called `jumpi`, but any name will work.
The user will be configured, so that it takes no password, but ssh-ing to it
will only work with private keys.

It's discouraged to use an existing user.

    adduser --system --shell /bin/sh --gecos 'ssh jumphost' \
    --group --disabled-password --home /home/jumpi jumpi

Next up, we will start the daemons, that are responsible for communicating
between each component and to present a nice web-ui.

    su - jump
    jumpi-web start
    jumpi-agent start

The Web-UI is available under 127.0.0.1:8080, so you will need to use
a SSH tunnel to connect to it.

### Alternative

You can also create your own configuration to start the agent and web webservers
(for example using nginx/uwsgi or apache/mod\_wsgi). This will allow you to choose
your own ports and bind the API to different ports. In addition, this also allows
you to have more than one UI and user per system.

We always suggest to bind the agent only on localhost (127.0.0.1) for security
reasons.

Both the Web-UI and the agent are fully compliant WSGI applications. For the agent
use `from jumpi.agent.api import app` (where app is the WSGI application). For
the Web-UI use `from jumpi.web import create_app` (create\_app is a function, that
returns a WSGI application).

If you choose non-standard ports to bind the applications to, you have to tell
them, on which port the agent can be accessed. Do this, by creating a file named
`jumpi-agent.cfg` in the home directory of the previously created user. In it
you can specify the agent host and port:

    [agent]
    host = 127.0.0.1
    port = 42000

## Configuration

Once you login to the Web-UI, configuration should be straight forward. Before
you edit anything though, you have to unlock the agent. This you can do under
"System".

## License

JumPi is licensed under the BSD License. See LICENSE for more information.

