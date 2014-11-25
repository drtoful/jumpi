Architecture
============

JumPi consists of three components:

+-------+-----------------------------------------------------------------------------------------------------------+
|Agent  |Provides an API to access the secure storage and the DB. Mainly used by the shell.                         |
+-------+-----------------------------------------------------------------------------------------------------------+
|Shell  |A reduced shell for the User to select the next hop. Is invoked when the User uses SSH to connect to JumPi.|
+-------+-----------------------------------------------------------------------------------------------------------+
|Web UI |Administrative UI to add/remove SSH targets and User.                                                      |
+-------+-----------------------------------------------------------------------------------------------------------+

To securely store passwords, SSH private keys and session replays, JumPi uses `PyVault`_ to do so.

.. _PyVault: https://github.com/drtoful/pyvault

Agent
-----

The agent acts as a link for the shell to the secure storage and DB information. The data can be accessed via the
:doc:`api`.

The reason for the agent is, that you do not want to give out the unlock passphrase for the secure storage to all
of your users. But this is needed in order to access the stored passwords. So the agent is unlocked once by the admin
in the Web UI and can then used by all users (confined within the shell).

The other reason is for access to the DB. Computing the sqlalchemy objects takes a very long time (~5s) on 
startup of a shell. To reduce this number we decided to pack neccessary DB access into the Agent and let
the shell access this information over the API.

You may be concerned, that all secure data is accessible via the Agent (which makes you wonder, why bother and
savely store the data anyway). You're absolutely right to be concerned. The fact is, when you gain access to the
server on which JumPi is running, while the Agent is unlocked, an adversary may gain access to all information.
However you have to note, that this is also true for say a `HSM`_ that is connected to a server (e.g. you could
sign certificates).

`PyVault`_ only secures the data from being copied, e.g. by a flaw in a piece of software that let's you dump 
filesystem content. It does not protect you, when the adversary gained access to the server via SSH (same as
for the HSM).

.. _HSM: http://en.wikipedia.org/wiki/Hardware_security_module

Shell
-----

The shell is invoked, when a User connects using its private key via SSH. It provides a limited set of
commands to be executed by the User (e.g. ssh or scp).

This behavior is achieved by using the ``authorized_keys`` configuration file within the openssh implementation,
to control, which command is executed for a key.
