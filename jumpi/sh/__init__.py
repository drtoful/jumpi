#-*- coding: utf-8 -*-

import os
import logging
import logging.handlers
import random
import hashlib

from jumpi.config import HOME_DIR

_filename = os.path.join(HOME_DIR, "log")
if not os.path.isdir(_filename):
    os.mkdir(_filename, 0700)
_filename = os.path.join(_filename, "sh.log")
_session = None

def get_session_id():
    global _session

    if not _session is None:
        return _session

    r = str(random.random())
    d = hashlib.md5()
    d.update(r)
    _session = d.hexdigest()[:8]

    return _session

_format = logging.Formatter(
    "%(asctime)s %(name)s level=%(levelname)s session="+ \
    get_session_id()+" %(message)s")
_handler = logging.handlers.RotatingFileHandler(_filename, 'a', 1*1024*1024, 10)
_handler.setFormatter(_format)

log = logging.getLogger('jumpi.sh')
log.setLevel(logging.INFO)
log.addHandler(_handler)

