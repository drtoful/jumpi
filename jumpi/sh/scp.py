#-*- coding: utf-8 -*-

import os
import sys
import re
import select
import time
import hashlib
import datetime

from jumpi.sh import log, HOME_DIR
from jumpi.db import Session, File

_copy_re = re.compile(
    "C(?P<mode>\d+) (?P<length>\d+) (?P<filename>.*)",
    re.IGNORECASE
)
_dir_re = re.compile(
    "D(?P<mode>\d+) \d+ (?P<dirname>.*)",
    re.IGNORECASE
)

class _PseudoSocket(object):
    def __init__(self, stdin, stdout):
        self.stdin = stdin
        self.stdout = stdout
        self.transport = None

    def send(self, data):
        self.stdout.flush()
        self.stdout.write(data)
        self.stdout.flush()
        return len(data)

    def recv(self, bufsize):
        data = self.stdin.read(bufsize)
        return data

    def tell(self):
        rlist, _, _ = select.select([self.stdin],[],[],0.1)
        return self.stdin in rlist

class _SCPException(Exception):
    pass

class _SCPServerInterface(object):
    def open(self, path, attr, flags):
        raise _SCPException("not implemented: open(%s)" % path)

    def mkdir(self, path, attr):
        raise _SCPException("not implemented: mkdir(%s)" % path)

class _SCPServer(object):
    CMD_OK = "\x00"
    CMD_WARN = "\x01\n"
    CMD_ERR = "\x02\n"

    def __init__(self, socket, si=_SCPServerInterface()):
        self.socket = socket
        self.si = si
        self._dirstack = []

    def _readline(self):
        data = []
        while True:
            c = self.socket.recv(1)
            if c == "\n":
                break
            if c == "\x00":
                break
            if c == "\xff":
                break
            data.append(c)
        return "".join(data)

    def _copy(self, line):
        # check consistency
        match = _copy_re.match(line)
        if match is None:
            self.socket.send(_SCPServer.CMD_ERR)
            return

        # acknowledge command and parse arguments
        self.socket.send(_SCPServer.CMD_OK)
        length = int(match.group('length'))
        filename = match.group('filename')

        # open file and write data from socket
        path = os.path.join("/".join(self._dirstack), filename)
        fp = self.si.open(path, 0, 0)
        bytes = 0
        while length > 0:
            step = (length, 4096)[length > 4096]
            data = self.socket.recv(step)
            fp.write(data)
            length -= len(data)
            bytes += len(data)
        fp.close()

        # send confirmation
        self.socket.send(_SCPServer.CMD_OK)

    def _directory(self, line):
        # check consistency
        match = _dir_re.match(line)
        if match is None:
            self.socket.send(_SCPServer.CMD_ERR)
            return

        # acknowledge command and parse arguments
        self.socket.send(_SCPServer.CMD_OK)
        dirname = match.group('dirname')

        # create directory and push onto directory stack
        self._dirstack.append(dirname)
        self.si.mkdir(dirname, 0)

    def receive(self):
        time.sleep(0.1) # make sure receiver is ready
        self.socket.send(_SCPServer.CMD_OK)
        time.sleep(0.1) # make sure receiver got answer
        while True:
            line = self._readline()
            if line is None or len(line) == 0:
                break

            if line.startswith("C"):
                self._copy(line)
            elif line.startswith("D"):
                self._directory(line)
            elif line.startswith("E"):
                self._dirstack = self._dirstack[:-1]
                self.socket.send(_SCPServer.CMD_OK)
            else:
                self.socket.send(_SCPServer.CMD_OK)

            time.sleep(0.1) # wait for data to appear
            if not self.socket.tell():
                break

class JumpiStorage(_SCPServerInterface):
    DATA_DIRECTORY = os.path.join(HOME_DIR, "data")

    class JumpiFile(object):
        def __init__(self, path, user):
            self.user = user

            # make sure data directory exists
            try:
                os.makedirs(JumpiStorage.DATA_DIRECTORY)
            except:
                pass

            # convert path to unique hash
            digest = hashlib.sha256();
            digest.update(str(user.id)+"::"+path)

            # open the file to read or write
            self.path = path
            self.file = os.path.join(
                JumpiStorage.DATA_DIRECTORY,
                digest.hexdigest()
            )
            self.fp = open(self.file, "w+")

            # stores file attributes
            self.len = 0
            self.has_written = False

        def write(self, data):
            self.fp.write(data)
            self.len += len(data)
            self.has_written = True

        def close(self):
            # close file descriptor
            self.fp.close()
            if not self.has_written:
                return

            # save file for user
            file = File(
                user_id = self.user.id,
                basename = self.path,
                filename = self.file,
                created = datetime.datetime.now()
            )
            session = Session()
            session.merge(file)
            session.commit()

            # log
            log.info("user='%s' id=%d has uploaded new file=%s with length=%d" %
                (self.user.fullname, self.user.id, self.file, self.len))


    def __init__(self, user):
        self.user = user

    def open(self, path, attr, flags):
        return JumpiStorage.JumpiFile(path, self.user)

    def mkdir(self, path, attr):
        pass

class SCPServer(object):
    def retrieve(self, user):
        socket = _PseudoSocket(sys.stdin, sys.stdout)
        si = JumpiStorage(user)
        server = _SCPServer(socket, si=si)
        server.receive()

    def send(self):
        pass
