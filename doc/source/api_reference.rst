API Reference
=============

``POST`` /vault/unlock
----------------------

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

+--------------------+---------------------------------------------+
| | passphrase       |A secret passphrase to unlock PyVault storage|
| | *string,required*|                                             |
+--------------------+---------------------------------------------+

**Response**

+---+--------------------------------+
|200|Vault unlocked                  |
+---+--------------------------------+
|201|Vault created and unlocked      |
+---+--------------------------------+
|202|Vault already is unlocked       |
+---+--------------------------------+
|403|Unlock failed (wrong passphrase)|
+---+--------------------------------+

**Example Request**

.. code-block:: bash

    http POST http://127.0.0.1:42000/vault/unlock passphrase=test

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 0
    Content-Type: application/json; charset=utf-8

``GET`` /vault/status
---------------------

Queries the `PyVault`_ storage to check its status.
**Parameters**

*no parameters*

**Response**

The response contains a JSON object with the following
keys:

+------------+------------------------------------------------+
| | locked   |``true`` if vault is locked, ``false`` otherwise|
| | *boolean*|                                                |
+------------+------------------------------------------------+

**Example Request**

.. code-block:: bash

    http GET http://127.0.0.1:42000/vault/status

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 17
    Content-Type: application/json; charset=utf-8

    {
        "locked": false
    }


``PUT`` /vault/store
--------------------

Store data within the `PyVault`_. JumPi uses this storage to securely store
passwords and private keys for SSH targets and session replays. You can
store arbitrary data within the store.

**Parameters**

+--------------------+-----------------------------------------------+
| | key              |The key under which to store data              |
| | *string,required*|                                               |
+--------------------+-----------------------------------------------+
| | value            |The data to store                              |
| | *string,required*|                                               |
+--------------------+-----------------------------------------------+

We recommend to store binary data as base64 encoded string.

**Response**

+---+---------------------+
|200|Data was stored      |
+---+---------------------+

**Example Request**

.. code-block:: bash

    http PUT http://127.0.0.1:42000/vault/store key=myid value="secret phrase"

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 0
    Content-Type: application/json; charset=utf-8

``GET`` /vault/retrieve
-----------------------

Retrieve previously stored data from the `PyVault`_. Can also be used
to retrieve data that was set by JumPi. See Parameters on how JumPi
has stored its data.

**Parameters**

+--------------------+-----------------------------------------------+
| | key              |The key under which the data was stored        |
| | *string,required*|                                               |
+--------------------+-----------------------------------------------+

You can used the ID of a SSH target to retrieve its password or private
keys to connect to it. The key for stored session replays is composed of
the user ID and the session ID (concatenated via "@").

**Response**

+---+---------------------+
|200|Data follows         |
+---+---------------------+

The response contains a JSON object with the following
keys:

+-----------+------------------------------------------------+
| | value   |Value of the provided key that is stored in the |
| | *string*|vault                                           |
+-----------+------------------------------------------------+

**Example Request**

.. code-block:: bash

    http GET http://127.0.0.1:42000/vault/retrieve key=myid

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 26
    Content-Type: application/json; charset=utf-8

    {
        "value": "secret phrase"
    }

``GET`` /target
---------------

**Parameters**

+--------------------+---------------------------------------------+
| | id               |The ID of the SSH target to retrieve         |
| | *string,required*|                                             |
+--------------------+---------------------------------------------+

The ID is a concatenation (with "@") of the username and the host of the target.

**Response**

+---+---------------------+
|200|Target data follows  |
+---+---------------------+

The response contains a JSON object which contains the 
following keys:

+------------+------------------------------------------------------------------+
| | id       |The ID of the SSH target                                          |
| | *string* |                                                                  |
+------------+------------------------------------------------------------------+
| | port     |The port to connect to                                            |
| | *integer*|                                                                  |
+------------+------------------------------------------------------------------+
| | type     |The type of the secret that is stored in the secure storage. Can  |
| | *string* |be one of the following:                                          |
|            |                                                                  |
|            |* password                                                        |
|            |* key                                                             |
+------------+------------------------------------------------------------------+

**Example Request**

.. code-block:: bash

    http GET http://127.0.0.1:42000/target id=root@example.com

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 58
    Content-Type: application/json; charset=utf-8

    {
        "id": "root@example.com", 
        "port": 22, 
        "type": "password"
    }


``GET`` /user/info
------------------

Get information for a User.

**Parameters**

+---------------------+---------------------------------------+
| | user              |User ID                                |
| | *integer,required*|                                       |
+---------------------+---------------------------------------+

**Response**

+---+-----------------+
|200|User data follows|
+---+-----------------+

The response contains a JSON object with the following
keys:

+------------------+---------------------------------------------------+
| | id             |The User ID (corresponds to the ID you queried for)|
| | *string*       |                                                   |
+------------------+---------------------------------------------------+
| | fullname       |The name of the User when created in the Web UI    |
| | *string*       |                                                   |
+------------------+---------------------------------------------------+
| | ssh_fingerprint|Fingerprint of the User's SSH key                  |
| | *string*       |                                                   |
+------------------+---------------------------------------------------+
| | time_added     |Date and Time the User was added in the Web UI     |
| | *date*         |                                                   |
+------------------+---------------------------------------------------+
| | time_lastaccess|Date and Time the User has connected via SSH       |
| | *date*         |                                                   |
+------------------+---------------------------------------------------+
| | twofactor      |'true' if the User has activated TwoFactor         |
| | *boolean*      |Authentication, false otherwise                    |
+------------------+---------------------------------------------------+

**Example Request**

.. code-block:: bash

    http GET http://127.0.0.1:42000/user/info user:=1

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 182
    Content-Type: application/json; charset=utf-8

    {
        "fullname": "John Doe", 
        "time_added": "2014-11-01 12:00:00", 
        "ssh_fingerprint": "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99", 
        "id": 1, 
        "time_lastaccess": "2014-11-01 12:00:00"
    }

``PATCH`` /user/info
--------------------

Updates some values in the DB for the User.

**Parameters**

+---------------------+---------------------------------------+
| | user              |User ID                                |
| | *integer,required*|                                       |
+---------------------+---------------------------------------+
| | time_lastaccess   |                                       |
| | *date,optional*   |                                       |
+---------------------+---------------------------------------+
| | twofactor         |                                       |
| | *boolean,optional*|                                       |
+---------------------+---------------------------------------+

Updates all provided optional parameters.

**Response**

+---+-------------------------------------------+
|200|Data has been updated                      |
+---+-------------------------------------------+

**Example Request**

.. code-block:: bash

    http PATCH http://127.0.0.1:42000/user/info user:=1 time_lastaccess="1970-01-01 00:00:00"

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 0
    Content-Type: application/json; charset=utf-8

``GET`` /user/permissions
-------------------------

Get a list of SSH targets that this User is allowed to access.

**Parameters**

+---------------------+---------------------------------------+
| | id                |User ID                                |
| | *integer,required*|                                       |
+---------------------+---------------------------------------+

**Response**

+---+-------------------+
|200|Target list follows|
+---+-------------------+

The response contains a list of JSON object with the following keys:

+------------+-------------------------------------------------+
| | id       |ID of this permission                            |
| | *integer*|                                                 |
+------------+-------------------------------------------------+
| | user_id  |The User that is allowed to access the SSH target|
| | *integer*|                                                 |
+------------+-------------------------------------------------+
| | target_id|ID of the SSH target                             |
| | *string* |                                                 |
+------------+-------------------------------------------------+

**Example Request**

.. code-block:: bash

    http GET http://127.0.0.1:42000/user/permissions user:=1

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 75
    Content-Type: application/json; charset=utf-8

    {
        "permissions": [
            {
                "id": 1, 
                "target_id": "root@example.com", 
                "user_id": 1
            }
        ]
    }

``GET`` /user/files
-------------------

Get a list of files that the User has access to on JumPi (i.e. the files that were
uploaded or downloaded using scp).

**Parameters**

+---------------------+---------------------------------------+
| | user              |User ID                                |
| | *integer,required*|                                       |
+---------------------+---------------------------------------+

**Response**

+---+-------------------+
|200|File list follows  |
+---+-------------------+

The response contains a list of JSON objects with the following keys:

+------------+-------------------------------------------------------+
| | filename |Filename as stored on JumPi                            |
| | *string* |                                                       |
+------------+-------------------------------------------------------+
| | basename |The original filename                                  |
| | *string* |                                                       |
+------------+-------------------------------------------------------+
| | user_id  |The User this file belongs to                          |
| | *integer*|                                                       |
+------------+-------------------------------------------------------+
| | created  |Date and Time the file was uploaded/downloaded to JumPi|
| | *date*   |                                                       |
+------------+-------------------------------------------------------+
| | size     |Size of the file in bytes                              |
| | *integer*|                                                       |
+------------+-------------------------------------------------------+

**Example Request**

.. code-block:: bash

    http GET http://127.0.0.1:42000/user/files user:=1

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 148
    Content-Type: application/json; charset=utf-8

    {
        "files": [
            {
                "basename": "file.txt", 
                "created": "2014-11-01 12:00:00", 
                "filename": "/home/jumpi/data/aabbccddee", 
                "size": 256, 
                "user_id": 1
            }
        ]
    }


``DELETE`` /file
----------------

**Parameters**

+---------------------+---------------------------------------+
| | filename          |The filename of the file to delete     |
| | *string,required* |                                       |
+---------------------+---------------------------------------+

**Response**

+---+---------------------+
|200|File deleted         |
+---+---------------------+

**Example Request**

.. code-block:: bash

    http DELETE http://127.0.0.1:42000/file filename="/home/jumpi/data/aabbccddee"

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 0
    Content-Type: application/json; charset=utf-8

``PUT`` /file
------------------------

**Parameters**

+---------------------+-------------------------------------------------------+
| | filename          |Absolute path to the file stored on JumPi              |
| | *string,required* |                                                       |
+---------------------+-------------------------------------------------------+
| | basename          |The original filename                                  |
| | *string,required* |                                                       |
+---------------------+-------------------------------------------------------+
| | user_id           |The User this file belongs to                          |
| | *integer,required*|                                                       |
+---------------------+-------------------------------------------------------+
| | created           |Date and Time the file was uploaded/downloaded to JumPi|
| | *date,required*   |                                                       |
+---------------------+-------------------------------------------------------+
| | size              |Size of the file in bytes                              |
| | *integer,required*|                                                       |
+---------------------+-------------------------------------------------------+

**Response**

+---+---------------------+
|200|Data stored          |
+---+---------------------+

**Example Request**

.. code-block:: bash

    http PUT http://127.0.0.1:42000/file user_id:=1 filename="/home/jumpi/data/aabbccddee" basename="file.txt" created="1970-01-01 00:00:00" size:=256

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 0
    Content-Type: application/json; charset=utf-8

``PUT`` /recording
------------------

Stores information about a new recording the DB.

**Note:** The replay data has to be stored seperately by using the ``PUT /vault/store`` API endpoint.

**Parameters**

+---------------------+----------------------------------------+
| | user_id           |The User involved in this session       |
| | *integer,required*|                                        |
+---------------------+----------------------------------------+
| | session_id        |Unique session ID                       |
| | *string,required* |                                        |
+---------------------+----------------------------------------+
| | duration          |Duration of the session in seconds      |
| | *integer,required*|                                        |
+---------------------+----------------------------------------+
| | width             |Width of the Client SSH window/terminal |
| | *integer,required*|                                        |
+---------------------+----------------------------------------+
| | height            |Height of the Client SSH window/terminal|
| | *integer,required*|                                        |
+---------------------+----------------------------------------+
| | time              |Date and Time when the session started  |
| | *date,required*   |                                        |
+---------------------+----------------------------------------+

**Response**

+---+---------------------+
|200|Data stored          |
+---+---------------------+

**Example Request**

.. code-block:: bash

    http PUT http://127.0.0.1:42000/recording user_id:=1 session_id="aabbccdd" duration:=120 width:=80 height:=24 time="1970-01-01 00:00:00"

**Example Response**

.. code-block:: http

    HTTP/1.0 200 OK
    Content-Length: 0
    Content-Type: application/json; charset=utf-8

