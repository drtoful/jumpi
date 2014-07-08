#-*- coding: utf-8 -*-

import functools
import hashlib
import re
import datetime
import base64

from flask import Blueprint, redirect, url_for, request
from jumpi.web.decorators import templated
from jumpi.db import Session, User

user = Blueprint("user", __name__)
get = functools.partial(user.route, methods=['GET'])
post = functools.partial(user.route, methods=['POST'])

_key_re = re.compile("ssh-rsa [a-z0-9+/=]+.*", re.IGNORECASE)
_title_re = re.compile("[^<>{}\[\]]+", re.IGNORECASE)
_id_re = re.compile("[0-9]+")

@get("/")
@templated("user.xhtml")
def index():
    session = Session()
    users = session.query(User).order_by(User.id)

    return dict(
        users = users
    )

@post("/add")
def add_key():
    def _calc_fingerprint(line):
        key = base64.b64decode(line.strip().split()[1].encode('ascii'))
        fp_plain = hashlib.md5(key).hexdigest()
        return ':'.join(a+b for a,b in zip(fp_plain[::2], fp_plain[1::2]))

    title = request.form.get("title", "")
    key = request.form.get("key", "")
    if _key_re.match(key) is None or _title_re.match(title) is None:
        return redirect(url_for('user.index'))

    user = User(
        fullname=title,
        ssh_key=key,
        ssh_fingerprint=_calc_fingerprint(key),
        time_added=datetime.datetime.now()
    )

    session = Session()
    session.add(user)
    session.commit()

    return redirect(url_for('user.index'))

@post("/delete")
def delete_key():
    id = request.form.get("id","")
    if _id_re.match(id) is None:
        return redirect(url_for('user.index'))

    session = Session()
    user = session.query(User).filter_by(id=id).first()
    if not user is None:
        session.delete(user)
        session.commit()

    return redirect(url_for('user.index'))

