#-*- coding: utf-8 -*-

import sys
import cmd
import paramiko
import StringIO
import shlex

# only works on unix
import termios
import tty

from jumpi.db import Session, TargetPermission
from jumpi.sh.agent import Agent
from jumpi.sh.scp import SCPServer
from jumpi.sh import log, get_session_id
from optparse import OptionParser

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

    def do_exit(self, line):
        return True

    def do_scp(self, line):
        try:
            # parse argument line to see if we need to
            # go into "server" mode
            args = shlex.split(line)
            parser = OptionParser()
            parser.add_option("-t", action="store_true") # source mode
            parser.add_option("-f", action="store_true") # sink mode
            # some default scp options that might get passed
            parser.add_option("-q", action="store_true")
            parser.add_option("-r", action="store_true")
            parser.add_option("-p", action="store_true")
            parser.add_option("-v", action="store_true")
            parser.add_option("-d", action="store_true")
            (options, args) = parser.parse_args(args)

            if options.t:
                scp = SCPServer()
                scp.retrieve(self.user)
                return False
            if options.f:
                scp = SCPServer()
                scp.send(self.user, args[0])
                return False
        except:
            log.error("session=%s unable to parse scp line \"%s\", \"%s\"" % (
                self.session, line, sys.exc_info()[0]))

        # otherwise we're in normal "client mode"
        return False

    def do_ls(self, line):
        for file in self.user.files:
            print file.basename, file.filename

    def do_ssh(self, line):
        target_id = line.split(" ", 1)[0].strip()
        session = Session()
        perm = session.query(TargetPermission).filter_by(
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

