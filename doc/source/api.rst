Agent API
=========

The Agent API is currently in a very unstable state and may change. 
You can use the API to access some information in the DB 
and the PyVault storage. You can only fully change entries in the DB via the admin GUI.

Examples are given as `httpie`_ commands.

The API is mainly used by the JumPi shell process to get access to DB information and
store/retrieve data from the secure storage.

You could use this API to use in scripts to open SSH connections without knowing
the password.

.. _httpie: http://httpie.org

Parameters
----------

API methods can contain required and optional parameters. Parameters can be a segment
of the request URL. If not, parameters should be encoded as JSON and passed with
a content-type ``application/json``:

.. code-block:: http

    POST /endpoint HTTP/1.0
    Content-Type: application/json

    {"argument": "value"}

Root Endpoint
-------------

By default you can access the agent under the following URL

.. code-block:: bash

    http POST http://127.0.0.1:42000

Note that the endpoint may differ, if you bound the agent
to a different port or IP.

Datatypes
---------

+---------+-------------------------------------------------------------------+
| string  | Arbitrary character stream                                        |
+---------+-------------------------------------------------------------------+
| boolean | ``true`` or ``false``                                             |
+---------+-------------------------------------------------------------------+
| integer | Any number                                                        |
+---------+-------------------------------------------------------------------+
| date    | Date and Time given as string in the format "YYYY-MM-DD HH:MM:SS" |
+---------+-------------------------------------------------------------------+


API Reference
-------------

.. toctree::
    :maxdepth: 2

    api_reference
