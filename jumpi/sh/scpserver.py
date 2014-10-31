#-*- coding: utf-8 -*-

#
# based on the works of James Bardin's pure python implementation
# of scp protcol. Licensed under GNU LGPL 2.1.
#
# See: https://github.com/jbardin/scp.py
#

import sys
import re
import os
import hashlib
import datetime
import shlex

from jumpi.sh import log, HOME_DIR
from jumpi.db import Session, File

_copy_re = re.compile(
    "C(?P<mode>\d{4}) (?P<length>\d+) (?P<filename>.*)",
    re.IGNORECASE
)
_dir_re = re.compile(
    "D(?P<mode>\d{4}) \d+ (?P<dirname>.*)",
    re.IGNORECASE
)

class StdSocket(object):
    class TimeoutError(Exception):
        pass

    def __init__(self):
        self.stdin = sys.stdin
        self.stdout = sys.stdout
        self.settimeout(1)

    def settimeout(self, seconds):
        self.timeout = seconds

    def readline(self):
        data = []
        while True:
            c = self.read(1)
            if c in (None, b"\n", b"\xff"):
                break
            data.append(c)

        return "".join(data)

    def read(self, bytes):
        import signal

        def handler(signum, frame):
            raise StdSocket.TimeoutError()

        signal.signal(signal.SIGALRM, handler)
        signal.alarm(5)

        try:
            result = sys.stdin.read(bytes)
            if len(result) == 0:
                raise StdSocket.TimeoutError()
        except StdSocket.TimeoutError as exc:
            result = None
        finally:
            signal.alarm(0)

        return result

    def write(self, data):
        self.stdout.flush()
        self.stdout.write(data)
        self.stdout.flush()

class JumpiFile(object):
    O_READ = "r"
    O_WRITE = "w"
    DATA_DIRECTORY = os.path.join(HOME_DIR, "data")

    def __init__(self, path, user, flag, mode):
        # make sure data directory exists
        try:
            os.makedirs(JumpiFile.DATA_DIRECTORY, mode=0700)
        except:
            pass

        # convert path to unique hash
        digest = hashlib.sha256()
        digest.update(str(user.id)+"::"+path)

        self.path = path
        self.file = os.path.join(
            JumpiFile.DATA_DIRECTORY,
            digest.hexdigest()
        )

        # open file descriptor
        if flag == JumpiFile.O_READ:
            self.fp = open(self.file, "rb")
            self.readonly = True
        elif flag == JumpiFile.O_WRITE:
            self.fp = open(self.file, "wb")
            self.readonly = False
        else:
            raise IOError("unknown flag")

        # save some file attributes
        self.len = 0
        self.mode = mode
        self.user = user

    def basename(self):
        return self.path.rsplit("/", 1)[-1]

    def length(self):
        self.fp.seek(0, 2)
        size = self.fp.tell()
        self.fp.seek(0, 0)
        return size

    def read(self, length):
        return self.fp.read(length)

    def write(self, data, length):
        self.fp.write(data)
        self.len += length

    def close(self):
        self.fp.close()
        if self.readonly:
            return

        # store file in db
        file = File(
            user_id = self.user.id,
            basename = self.path,
            filename = self.file,
            created = datetime.datetime.now()
        )
        session = Session()
        session.merge(file)
        session.commit()

        # change filemode
        #TODO

class SCPException(Exception):
    def __init__(self, msg=""):
        self.msg = msg

class SCPServer(object):
    CMD_OK = b"\x00"
    CMD_WARN = b"\x01\n"
    CMD_ERR = b"\x02\n"


    def __init__(self, socket, user):
        self.socket = socket
        self.user = user
        self._dirstack = []

    def _confirm(self):
        msg = self.socket.read(1)
        if msg is None:
            raise SCPException("No or invalid response")

        if msg[0:1] == SCPServer.CMD_OK:
            return

        msg = self.socket.readline()
        raise SCPException(msg)

    def _recv_file(self, line):
        match = _copy_re.match(line)
        if match is None:
            raise SCPException("invalid command: %s" % line)

        self.socket.write(SCPServer.CMD_OK)
        size = int(match.group('length'))
        filename = match.group('filename')

        path = os.path.join("/".join(self._dirstack), filename)
        fp = JumpiFile(path, self.user, JumpiFile.O_WRITE, 0600)

        while size > 0:
            step = (size, 4096)[size > 4096]
            data = self.socket.read(step)
            if data is None:
                raise SCPException("Error receiving, socket timeout")
            fp.write(data, len(data))
            size -= len(data)

        self._confirm()
        fp.close()

    def _recv_pushd(self, line):
        match = _dir_re.match(line)
        if match is None:
            raise SCPException("invalid command: %s" % line)

        dirname = match.group('dirname')
        self._dirstack.append(dirname)

    def _recv_popd(self, line):
        if line != "E":
            raise SCPException("invalid command: %s" % line)

        self._dirstack = self._dirstack[:-1]

    def _send_file(self, path, recursive=False):
        # send recursive directories if wanted
        if recursive:
            dirs = path.split("/")[:-1]
            for d in dirs:
                self._send_pushd(d)

        # open file
        fp = JumpiFile(path, self.user, JumpiFile.O_READ, 0)

        # send command, replace \n with control sequence
        # \^J (like openssh)
        size = fp.length()
        self.socket.write("C0600 %d %s\n" % (
            size, fp.basename().replace('\n', '\\^J')
        ))
        self._confirm()

        # send data
        while size > 0:
            step = (size, 4096)[size > 4096]
            data = fp.read(step)
            self.socket.write(data)
            size -= len(data)

        # close up channel
        self.socket.write(SCPServer.CMD_OK)
        self._confirm()
        fp.close()

        # close recursive directories
        if recursive:
            dirs = path.split("/")[:-1]
            for _ in xrange(0, len(dirs)):
                self._send_popd()

    def _send_pushd(self, path):
        # send command, replace \n with control sequence
        # \^J (like openssh)
        self.socket.write("D0700 0 %s\n" % path.replace('\n', '\\^J'))
        self._confirm()

    def _send_popd(self):
        self.socket.write("E\n")
        self._confirm()

    def receive(self):
        commands = {b'C': self._recv_file,
                    b'T': lambda x: None,
                    b'D': self._recv_pushd,
                    b'E': self._recv_popd}

        while True:
            self.socket.write(SCPServer.CMD_OK)
            msg = self.socket.readline()
            if msg in (None, ""):
                break

            code = msg[0:1]
            try:
                commands[code](msg)
            except KeyError:
                raise SCPException("Unknown command: %s" % str(msg).strip())

    def send(self, path, recursive=False):
        # convert '*' in path to regex equivalent '.*'
        path = path.replace('*', '.*')
        match_re = re.compile(path, re.IGNORECASE)

        # process all files, matching the given path
        for file in self.user.files:
            match = match_re.match(file.basename)
            if not match is None:
                self._send_file(file.basename, recursive)

def scp_receive(user, session):
    try:
        socket = StdSocket()
        server = SCPServer(socket, user)
        server.receive()
    except SCPException as exc:
        log.error("session=%s scp recv failure msg=\"%s\"" % (session, exc.msg))

def scp_send(user, session, path, recursive=False):
    try:
        socket = StdSocket()
        server = SCPServer(socket, user)
        server.send(path, recursive)
    except SCPException as exc:
        log.error("session=%s scp send failure msg=\"%s\"" % (session, exc.msg))

def scp_parse_command(command):
    recursive_re = re.compile("\-[^r\-\s]*r[^r\-\s]*")
    from_re = re.compile("\-[^f\-\s]*f[^f\-\s]*")
    to_re = re.compile("\-[^t\-\s]*t[^t\-\s]*")
    args = shlex.split(command)

    return {
        "r": not recursive_re.search(command) is None,
        "t": not to_re.search(command) is None,
        "f": not from_re.search(command) is None,
        "path": args[-1]
    }

