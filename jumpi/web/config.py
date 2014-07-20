#-*- coding: utf-8 -*-

import os

class JumpiConfig(object):
    SECRET_KEY = os.urandom(64)
    DEBUG = False
    MODULES = [
        ("jumpi.web.base.base", "/"),
        ("jumpi.web.user.user", "/user"),
        ("jumpi.web.system.system", "/system"),
        ("jumpi.web.target.target", "/target"),
        ("jumpi.web.tunnel.tunnel", "/tunnel")
    ]

