#-*- coding: utf-8 -*-

import os
import logging
import random
import hashlib

_filename = os.path.expanduser("~")
_filename = os.path.join(_filename, "log")
if not os.path.isdir(_filename):
    os.mkdir(_filename, 0700)
_filename = os.path.join(_filename, "sh.log")

_format = logging.Formatter(
    "%(asctime)s %(name)s level=%(levelname)s %(message)s")
_handler = logging.FileHandler(_filename)
_handler.setFormatter(_format)

log = logging.getLogger('jumpi.sh')
log.setLevel(logging.INFO)
log.addHandler(_handler)

def get_session_id():
    r = str(random.random())
    d = hashlib.md5()
    d.update(r)
    return d.hexdigest()[:8]


