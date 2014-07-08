#-*- coding: utf-8 -*-

import os

class JumpiConfig(object):
    SECRET_KEY = os.urandom(64)
    DEBUG = False
    MODULES = [
        ("jumpi.web.base.base", "/"),
        ("jumpi.web.user.user", "/user"),
    ]

