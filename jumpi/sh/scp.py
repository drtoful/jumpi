#-*- coding: utf-8 -*-

import os
import sys
import re
import select
import time

from jumpi.sh import log

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
        return select.select([self.stdin,],[],[],0.0)[0]

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
            data.append(c)
            if c == "\x00":
                break
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

    def start(self):
        self.socket.send(_SCPServer.CMD_OK)
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
    class JumpiFile(object):
        def write(self, data):
            pass

        def close(self):
            pass

    def open(self, path, attr, flags):
        return JumpiStorage.JumpiFile()

    def mkdir(self, path, attr):
        pass

class SCP(object):
    def retrieve(self):
        socket = _PseudoSocket(sys.stdin, sys.stdout)
        si = JumpiStorage()
        server = _SCPServer(socket, si=si)
        server.start()

    def send(self):
        pass
