#-*- coding: utf-8 -*-

import os
import logging
import logging.handlers
import random
import hashlib

from flask import Flask

try:
    import pwd
    HOME_DIR = pwd.getpwuid(os.getuid()).pw_dir
except:
    HOME_DIR = os.path.expanduser("~")

_filename = os.path.join(HOME_DIR, "log")
if not os.path.isdir(_filename):
    os.mkdir(_filename, 0700)
_filename = os.path.join(_filename, "agent.log")

_format = logging.Formatter(
    "%(asctime)s %(name)s level=%(levelname)s %(message)s")
_handler = logging.handlers.RotatingFileHandler(_filename, 'a', 1*1024*1024, 10)
_handler.setFormatter(_format)

log = logging.getLogger('jumpi.agent')
log.setLevel(logging.INFO)
log.addHandler(_handler)

def get_session_id():
    r = str(random.random())
    d = hashlib.md5()
    d.update(r)
    return d.hexdigest()[:8]

def create_app():
    from jumpi.agent.api import app as api_app

    webapp = Flask(__name__)
    webapp.register_blueprint(api_app)

    return webapp
