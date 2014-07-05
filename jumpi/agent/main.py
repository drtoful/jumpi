#-*- coding: utf-8 -*-

import os

from jumpi.agent.api import app
from jumpi.app import DaemonApp

class Main(DaemonApp):
    def __init__(self):
        DaemonApp.__init__(self)
        home = os.path.expanduser("~")
        self.pidfile = os.path.join(home, "jumpi-agent.pid")

    def start(self):
        app.run(host="127.0.0.1", port=42000)

