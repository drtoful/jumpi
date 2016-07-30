#-*- coding: utf-8 -*-

import os

class JumpiConfig(object):
    ROOT_ENDPOINT = "/"
    DEBUG = False
    SECRET_KEY = os.urandom(64)
    SESSION_TIMEOUT = 3600

    MODULES = [
        ("jumpi.ui.uibp", "/"),
    ]

    # configuration for jumpi control port
    JUMPI_HOST = 'localhost'
    JUMPI_PORT = 4200

    def __init__(self):
        # update config variables from environment
        for key in dir(self):
            if key.startswith("MODULES"):
                continue

            if 65 <= ord(key[0]) <= 90 and os.environ.has_key(key):
                # check if old value is an integer
                value = getattr(self, key)
                if isinstance(value, (int, float)):
                    setattr(self, key, int(os.environ.get(key)))
                else:
                    setattr(self, key, os.environ.get(key))
