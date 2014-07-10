#-*- coding: utf-8 -*-

import functools
import SecureString
import re

from flask import Blueprint, redirect, url_for, request
from jumpi.web.decorators import templated
from jumpi.db import Session, Target

target = Blueprint("target", __name__)
get = functools.partial(target.route, methods=['GET'])
post = functools.partial(target.route, methods=['POST'])

_username_re = re.compile("^[a-z][a-z0-9_\-]+", re.IGNORECASE)
_hostname_re = re.compile(
    "^(?:[A-Z0-9](?:[A-Z0-9-]{0,61}[A-Z0-9])?\.)+[A-Z]{2,6}$",
    re.IGNORECASE
)
_ip_re = re.compile("[0-9\.]+")
_port_re = re.compile("[0-9]+")

@get("/")
@templated("target.xhtml")
def index():
    session = Session()
    targets = session.query(Target).order_by(Target.id)

    return dict(targets = targets)

@post("/add")
def add_target():
    username = request.form.get("username", "")
    hostname = request.form.get("target", "")
    port = request.form.get("port", "")
    type = request.form.get("type", "password")
    key = request.form.get("key", "")

    if _username_re.match(username) is None or \
        (_hostname_re.match(hostname) is None and \
        _ip_re.match(hostname) is None) or _port_re.match(port) is None:
        return redirect(url_for('target.index'))

    target = Target(
        id="%s@%s" % (username, hostname),
        type = type,
        port = int(port)
    )

    session = Session()
    session.add(target)
    session.commit()

    return redirect(url_for('target.index'))
