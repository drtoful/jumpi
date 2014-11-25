Admin Manual
============

Service Installation
--------------------

**uwsgi**

We recommend installing uwsgi from the pypi repositories instead of the version
provided by the distribution. To do so use the following command:

::

    pip install uwsgi

The provided configuration files in :doc:`reference` can be used directly
with the `Emperor <http://uwsgi-docs.readthedocs.org/en/latest/Emperor.html>`_ mode of uwsgi.
Copy the sample configuration files to a directory (for example */etc/uwsgi*).

You can then start the uwsgi Emperor with the following command:

::

    uwsgi --emperor /etc/uwsgi

See also `this article <uwsgi-docs.readthedocs.org/en/latest/Upstart.html>`_ to see
how you can configure uwsgi to be started with upstart.

**nginx**

Install nginx from your distribution's repository. For example:

::

    aptitude install nginx

Copy then the provided sample configuration file from :doc:`reference` into
*/etc/nginx/sites-available*.

The Web-UI will be bound to port 443 (HTTPS) so you will need a certificate
plus key. Change this part of the nginx configuration to the correct path
of the certificate.

Enable then the configuration by linking (*ln*) the config in sites-enabled and
then restart nginx.

Virtual Environment
-------------------

You can install JumPi within a python `Virtual Environment`_. If you do so however, you will
need to create a wrapper script, that will invoke the JumPi shell under the correct 
path. Adapt the following snippet and store it under ``/usr/local/bin/jumpish``. Make sure
it is executable by your JumPi user and that the PATH is correctly setup.

.. code-block:: bash

    #!/bin/bash
    . /path/to/venv/bin/activate
    /path/to/venv/bin/jumpish

.. _Virtual Environment: http://virtualenv.readthedocs.org/en/latest/
