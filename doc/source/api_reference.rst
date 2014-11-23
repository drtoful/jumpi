API Reference: Vault
====================

``POST`` /unlock
----------------

JumPi uses `PyVault`_ to securely store passwords and RSA private
keys to access SSH targets. The storage needs to be unlocked before
it can be used. If the storage does not yet exist, one will be
created.

Specific vault parameters that change certain cryptographic
properties can only be controlled via configuration. See
:doc:`reference` for a description of these configuration
parameters.

.. _PyVault: https://github.com/drtoful/pyvault

**Parameters**

+-----------------+---------------------------------------------+
|passphrase       |A secret passphrase to unlock PyVault storage|
|*string,required*|                                             |
+-----------------+---------------------------------------------+

**Response**

+---+--------------------------------+
|200|Vault created/unlocked          |
+---+--------------------------------+
|403|Unlock failed (wrong passphrase)|
+---+--------------------------------+
|500|Internal Server Error           |
+---+--------------------------------+

**Example Request**

.. code-block:: bash

    http POST http://127.0.0.1:42000/unlock 

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 0
    Content-Type: text/html; charset=utf-8
    Date: Sun, 23 Nov 2014 21:38:34 GMT
    Server: Werkzeug/0.9.6 Python/2.7.6

``GET`` /ping
-------------

Queries the `PyVault`_ storage to check if the storage is locked
or not.

**Parameters**

*no parameters*

**Response**

The response contains a JSON object with the following
keys:

+---------+------------------------------------------------+
|pong     |``true`` if vault is locked, ``false`` otherwise|
|*boolean*|                                                |
+---------+------------------------------------------------+

**Example Request**

.. code-block:: bash

    http GET http://127.0.0.1:42000/ping

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 15
    Content-Type: text/html; charset=utf-8
    Date: Sun, 23 Nov 2014 21:44:58 GMT
    Server: Werkzeug/0.9.6 Python/2.7.6

    {"pong": false}

``GET`` /retrieve
-----------------

**Parameters**

**Response**

**Example Request**

**Example Response**

``PUT`` /store
--------------

**Parameters**

**Response**

**Example Request**

**Example Response**

API Reference: Target
=====================

``GET`` /target
---------------

**Parameters**

**Response**

**Example Request**

**Example Response**

API Reference: User
===================

``GET`` /user/{id}/info
-----------------------

**Parameters**

**Response**

**Example Request**

**Example Response**

``POST`` /user/{id}/info
------------------------

**Parameters**

**Response**

**Example Request**

**Example Response**

``GET`` /user/{id}/targets
--------------------------

**Parameters**

**Response**

**Example Request**

**Example Response**

``GET`` /user/{id}/files
------------------------

**Parameters**

**Response**

**Example Request**

**Example Response**

``DELETE`` /user/{id}/files
---------------------------

**Parameters**

**Response**

**Example Request**

**Example Response**

``PUT`` /user/{id}/files
------------------------

**Parameters**

**Response**

**Example Request**

**Example Response**

``PUT`` /user/{id}/recording
----------------------------

**Parameters**

**Response**

**Example Request**

**Example Response**
