#-*- coding: utf-8 -*-

from jumpi.sh.scpserver import SCPServer, SCPException

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
    try:
        socket = ChannelSocket(channel)
        server = SCPServer(socket, user, session)

        channel.exec_command("scp -r -t %s" % path)
        server._confirm() # wait for OK of remote
        server.send(file, True)
    except SCPException as exc:
        log.error("session=%s scp client send failure " \
            "msg=\"%s\"" % (session, exc.msg))

def scpc_receive(channel, path, user, session):
    try:
        socket = ChannelSocket(channel)
        server = SCPServer(socket, user, session)

        channel.exec_command("scp -r -f %s" % path)
        server.receive()
    except SCPException as exc:
        log.error("session=%s scp client recv failure " \
            "msg=\"%s\"" % (session, exc.msg))

