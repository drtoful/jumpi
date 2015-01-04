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

TwoFactor Authentication
------------------------

You can setup JumPi to allow users to enable two factor authentication. The user can
select between multiple methods:

* TOTP/HOTP according to RFC 6238 (GoogleAuthenticator compatible)
* Yubico `YubiKey`_ OTP

.. _YubiKey: https://www.yubico.com/products/yubikey-hardware/

**TOTP/HOTP according to RFC 6238**

In order to enable this authentication method you need to fullfill at least the
following dependencies:

* `pyotp`_ >= 1.3.0
* `qrcode`_ >= 5.1

You can also resolve these dependencies with the following command within the source
directory:

::

    easy_install . jumpi[with_otp_google]


.. _pyotp: https://github.com/nathforge/pyotp
.. _qrcode: https://github.com/lincolnloop/python-qrcode

**Yubico YubiKey OTP**

In order to enable this authentication method you need to fullfill at least the
following dependencies:

* `yubico_client`_ >= 1.9.1

You can also resolve these dependencies with the following command within the source
directory:

::

    easy_install . jumpi[with_otp_yubico]

If you do not have your own validation server running and plan to use the public
cloud servers, you have to register yourself on the following page: https://upgrade.yubico.com/getapikey/.

After that you need to configure the client id and the secret key in the config file:

::

    [yubico]
    api_clientid = xxxx
    api_secret = yyyy

.. _yubico_client: https://github.com/Kami/python-yubico-client

Session Recording
-----------------

You can configure JumPi to process session recordings, so that you can watch them using a Javascript
in your browser. In order to enable this feature, you need to install at least these
dependencies:

* `pyte`_ >= 0.4.8

You can also resolve these dependencies with the following command within the source
directory:

::

    easy_install . jumpi[with_pyte]
 
Note, that enabling this will probably slow down your connection and use a bit of resources.

.. _pyte: http://pyte.readthedocs.org/en/latest/

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
