#-*- coding: utf-8 -*-

import os
import sys
import cmd
import StringIO
import re

# only works on unix
import termios
import tty

from jumpi.sh.agent import File, Vault
from jumpi.sh import log, get_session_id
from jumpi.sh.scpserver import scp_receive, scp_send, scp_parse_command
from jumpi.sh.scpclient import scpc_receive, scpc_send

_scp_from_re = re.compile(
    "(?P<file>.*) (?P<target>[^@]*@[^:]*):(?P<path>.*)"
)
_scp_to_re = re.compile(
    "(?P<target>[^@]*@[^:]*):(?P<path>.*?) (?P<file>.*)"
)

class JumpiShell(cmd.Cmd):
    def __init__(self, user, **kwargs):
        cmd.Cmd.__init__(self, **kwargs)

        self.session = get_session_id()
        self.user = user
        self.systems = [x.target_id for x in user.target_permissions]
        log.info("session=%s user='%s' id=%s - session opened",
            self.session, self.user.fullname, self.user.id)

    def _shell(self, chan):
        import select
        import paramiko

        oldtty = termios.tcgetattr(sys.stdin)
        try:
            tty.setraw(sys.stdin.fileno())
            tty.setcbreak(sys.stdin.fileno())
            chan.settimeout(0.0)

            while True:
                r, w, e = select.select([chan, sys.stdin], [], [])
                if chan in r:
                    try:
                        x = paramiko.py3compat.u(chan.recv(1024))
                        if len(x) == 0:
                            sys.stdout.write("\r\n*** EOF\r\n")
                            break
                        sys.stdout.write(x)
                        sys.stdout.flush()
                    except socket.timeout:
                        pass

                if sys.stdin in r:
                    x = sys.stdin.read(1)
                    if len(x) == 0:
                        break
                    chan.send(x)
        finally:
            termios.tcsetattr(sys.stdin, termios.TCSADRAIN, oldtty)

    def _open_ssh_client(self, target_id):
        import paramiko

        perm = [x for x in self.user.target_permissions
            if x.user_id == self.user.id and target_id == target_id]

        if len(perm) == 0:
            log.error("session=%s target='%s' - access denied, user " \
                "has no permission to access this target" % (
                self.session, target_id))
            print "Permission denied!"
            return

        vault = Vault()
        secret = vault.retrieve(target_id)
        perm = perm[0]

        if secret is None:
            log.error("session=%s target='%s' - access denied, could " \
                "not load secret" % (self.session, target_id))
            print "Permission denied!"
            return

        client = paramiko.SSHClient()
        username, hostname = target_id.split("@",1)
        client.set_missing_host_key_policy(paramiko.client.WarningPolicy())
        if perm.target.type == "key":
            keyio = StringIO.StringIO(secret)
            pkey = paramiko.RSAKey.from_private_key(keyio)
            client.connect(port = perm.target.port, username = username,
                hostname = hostname, pkey = pkey)
        else:
            client.connect(port = perm.target.port, username = username,
                hostname = hostname, password = secret)

        return client

    def do_exit(self, line):
        return True

    def do_scp(self, line):
        log.info("session=%s invoked 'scp %s'" % (self.session, line))

        # client mode
        match = _scp_from_re.match(line)
        if not match is None:
            client = self._open_ssh_client(match.group('target'))
            if client is None:
                return False
            channel = client._transport.open_session()
            channel.settimeout(5)

            scpc_send(channel, match.group('file'), match.group('path'),
                self.user, self.session)
            channel.close()
            return False

        match = _scp_to_re.match(line)
        if not match is None:
            client = self._open_ssh_client(match.group('target'))
            if client is None:
                return False
            channel = client._transport.open_session()
            channel.settimeout(5)

            scpc_receive(channel, match.group('path'), self.user, self.session)
            channel.close()

            # update state in filelist of user
            self.user.load()

            return False

        # server mode
        opts = scp_parse_command(line)
        if opts["t"] and opts["f"]:
            log.error("session=%s sink and source mode requested")
            return False
        if opts["t"]:
            scp_receive(self.user, self.session)
            return False
        if opts["f"]:
            scp_send(self.user, self.session, opts["path"], opts["r"])
            return False

        log.error("session=%s unable to parse scp line" % self.session)

    def do_rm(self, line):
        for file in self.user.files:
            if file.basename == line:
                log.info("session=%s removing file='%s'" % (
                    self.session, file.basename))

                # remove from filesystem
                if os.path.isfile(file.filename):
                    os.remove(file.filename)

                # remove from db and refresh user object
                file = File(file.filename)
                file.delete()
                self.user.load()

                return False

    def do_ls(self, line):
        def _pretify_size(size):
            extension = ""
            num = size

            if num > 1024:
                num = round(num / 1024.0, 1)
                extension = "K"
            if num > 1024:
                num = round(num / 1024.0, 1)
                extension = "M"
            if num > 1024:
                num = round(num / 1024.0, 1)
                extension = "G"

            return ("%d%s" % (num, extension)).rjust(7)

        for file in self.user.files:
            print "%s %s\t%s" % (file.created, _pretify_size(file.size),
                file.basename)

    def do_ssh(self, line):
        target_id = line.split(" ", 1)[0].strip()

        client = self._open_ssh_client(target_id)
        if client is None:
            return False
        channel = client.invoke_shell()
        log.info("session=%s target='%s' - interactive shell invoked" % (
            self.session, target_id))
        self._shell(channel)
        channel.close()

    def do_2fa(self, line):
        from jumpi.sh.twofactor import TwoFactor

        type = line.split(" ",1)[0].strip()
        twofac = TwoFactor(self.user)

        if type == "remove":
            if twofac.remove():
                print "TwoFactor Authentication deactivated"
            return False

        if self.user.need_otp():
            print "Two Factor Authentication already setup"
            return False

        result = twofac.setup(type)
        if result:
            print "TwoFactor Authentication successfully setup!"
        else:
            print "Error in TwoFactor Authentication setup, try again"

    def complete_rm(self, text, line, start_index, end_index):
        if text:
            return [
                file.basename for file in self.user.files
                if file.basename.startswith(text)
            ]
        else:
            return self.user.files

    def complete_ssh(self, text, line, start_index, end_index):
        if text:
            return [
                address for address in self.systems
                if address.startswith(text)
            ]
        else:
            return self.systems

