Installation and Upgrade Manual
===============================

Source
------

In order to build some dependencies you might need to install additional
packages from your distribution's repository.

::

    aptitude install build-essential python-dev libffi-dev libssl-dev

JumPi can be installed from the sources directly by calling the installation
routine from the console:

::

    python setup.py install

We suggest using `virtualenv <http://virtualenv.readthedocs.org/en/latest/>`_.

You can remove the packages again after installation. They're just needed
to build the dependencies.

Dependencies
------------

The installation routine will automatically resolve all dependencies currently
needed to build and run JumPi. See README.md for full list of the
dependencies.

First-Time Configuration
------------------------

When you are installing JumPi for the first time, you need to follow these
steps:

1. Create a new underprivileged user, under which all the services will
   run. You can name this user whatever you want (default: jumpi). Make
   sure, that you disable the password for this user, so that only passwordless
   authentication with SSH keys will work.

::

   adduser --system --shell /bin/sh --gecos 'ssh jumphost' \
   --group --disabled-password --home /home/jumpi jumpi

2. Login as the created user by using 'su'

3. Initialize the DB

::

   jumpidb-create

Upgrade
-------

If you are upgrading JumPi from a previous version, you may need to upgrade the DB
to fullfill the current schema. You can do this by issuing the following
command:

::

    jumpidb-upgrade

Note, that you need to this as the user, under which you run JumPi (default: jumpi)

Services
--------

JumPi needs two services to be up and running in order to run correctly. These two
services are full `WSGI <http://wsgi.readthedocs.org/en/latest/>`_ compatible 
applications, and can thus be integrated in a multitude of ways.

We encourage the use of `uwsgi <https://uwsgi-docs.readthedocs.org/en/latest/>`_ and
`nginx <http://wiki.nginx.org/Main>`_. See :doc:`reference` for sample configuration
files for uwsgi and nginx.

You can of course use any other way that you might be familiar with to setup a WSGI
application. See also :doc:`admin`.
