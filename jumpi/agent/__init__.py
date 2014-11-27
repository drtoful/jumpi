#-*- coding: utf-8 -*-

import os
import logging
import logging.handlers
import random
import hashlib

from flask import Flask
from jumpi.config import HOME_DIR
from jumpi.agent.utils import compose_json_response

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

    # override default error handlers
    @webapp.errorhandler(404)
    def page_not_found(error):
        return compose_json_response(404, error="page_not_found",
            error_long="The page you requested could not be found")

    @webapp.errorhandler(405)
    def method_not_allowed(error):
        return compose_json_response(405, error="method_not_allowed",
            error_long="This HTTP method is not allowed on this endpoint")

    @webapp.errorhandler(500)
    def server_error(error):
        return compose_json_response(500, error="server_error",
            error_long="Something unforseen happened on the server!")

    return webapp
