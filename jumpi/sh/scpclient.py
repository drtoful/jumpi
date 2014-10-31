#-*- coding: utf-8 -*-

from jumpi.sh.scpserver import SCPServer, SCPException
from jumpi.sh import log

class ChannelSocket(object):
    def __init__(self, channel):
        self.channel = channel

    def settimeout(self, seconds):
        self.channel.settimeout(seconds)

    def readline(self):
        data = []
        while True:
            c = self.read(1)
            if c in (None, b"\n", b"\xff"):
                break
            data.append(c)

        return "".join(data)

    def read(self, bytes):
        if self.channel.closed:
            return None

        try:
            return self.channel.recv(bytes)
        except SocketTimeout:
            return None

    def write(self, data):
        self.channel.sendall(data)


def scpc_send(channel, file, path, user, session):
    def _callback(filename, size):
        print "uploaded %s (%d bytes)" % (filename, size)

    try:
        socket = ChannelSocket(channel)
        server = SCPServer(socket, user, session)

        channel.exec_command("scp -r -t %s" % path)
        server._confirm() # wait for OK of remote
        server.send(file, recursive=True, callback=_callback)
    except SCPException as exc:
        log.error("session=%s scp client send failure " \
            "msg=\"%s\" file=\"%s\"" % (session, exc.msg, file))

def scpc_receive(channel, path, user, session):
    def _callback(filename, size):
        print "downloaded %s (%d bytes)" % (filename, size)

    try:
        socket = ChannelSocket(channel)
        server = SCPServer(socket, user, session)

        channel.exec_command("scp -r -f %s" % path)
        server.receive(callback=_callback)
    except SCPException as exc:
        log.error("session=%s scp client recv failure " \
            "msg=\"%s\" file=\"%s\"" % (session, exc.msg, file))

