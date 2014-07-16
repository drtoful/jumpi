#-*- coding: utf-8 -*-

import sys
import cmd
import paramiko
import StringIO

# only works on unix
import termios
import tty

from jumpi.db import Session, Permission
from jumpi.sh.agent import Agent
from jumpi.sh import log, get_session_id

class JumpiShell(cmd.Cmd):
    def __init__(self, user, **kwargs):
        cmd.Cmd.__init__(self, **kwargs)

        self.session = get_session_id()
        self.user = user
        self.systems = [x.target_id for x in user.permissions]
        log.info("session=%s user='%s' id=%s - session opened",
            self.session, self.user.fullname, self.user.id)

    def _shell(self, chan):
        import select

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

    def do_ssh(self, line):
        target_id = line.split(" ", 1)[0].strip()
        session = Session()
        perm = session.query(Permission).filter_by(
            user_id=self.user.id, target_id=target_id).first()

        if perm is None:
            log.error("session=%s target='%s' - access denied, user " \
                "has no permission to access this target" % (
                self.session, target_id))
            print "Permission denied!"
            return

        a = Agent()
        secret = a.retrieve(target_id)

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
        channel = client.invoke_shell()
        log.info("session=%s target='%s' - interactive shell invoked" % (
            self.session, target_id))
        self._shell(channel)
        channel.close()

    def complete_ssh(self, text, line, start_index, end_index):
        if text:
            return [
                address for address in self.systems
                if address.startswith(text)
            ]
        else:
            return self.systems

