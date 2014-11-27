#-*- coding: utf-8 -*-

import os
import functools
import hashlib
import re
import datetime
import base64
import json
import struct

from flask import Blueprint, redirect, url_for, request, make_response
from jumpi.web.decorators import templated, authenticated, jsonr
from jumpi.config import HOME_DIR
from jumpi.db import Session, User
from jumpi.sh.agent import Vault

user = Blueprint("user", __name__)
get = functools.partial(user.route, methods=['GET'])
post = functools.partial(user.route, methods=['POST'])

_key_re = re.compile("ssh-[rd]sa [a-z0-9+/=]+.*", re.IGNORECASE)
_title_re = re.compile("[^<>{}\[\]]+", re.IGNORECASE)
_id_re = re.compile("[0-9]+")

def _recompute_authorized_keys():
    dir = os.path.join(HOME_DIR, ".ssh")
    file = os.path.join(dir, "authorized_keys")

    if not os.path.isdir(dir):
        os.makedirs(dir, 0700)

    session = Session()
    keys = session.query(User).all()

    with open(file, "w") as fp:
        print >>fp, "### autogenerated by JumPi, DO NOT EDIT"
        for key in keys:
            tunnels = ",".join(["permitopen=\"%s:%d\"" % \
                (x.tunnel.destination, x.tunnel.port)
                for x in key.tunnel_permissions])
            if len(key.tunnel_permissions) > 0:
                tunnels = ","+tunnels

            print >>fp, """command="jumpish %d",""" % (key.id) \
                +"""no-port-forwarding,no-X11-forwarding,""" \
                +"""no-agent-forwarding%s %s""" % (tunnels, key.ssh_key)

    os.chmod(file, 0600)

@get("/")
@authenticated
@templated("user.xhtml")
def index():
    session = Session()
    users = session.query(User).order_by(User.id)

    return dict(
        users = users
    )

@post("/add")
@authenticated
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

    _recompute_authorized_keys()

    return redirect(url_for('user.index'))

@post("/delete")
@authenticated
def delete_key():
    id = request.form.get("id","")
    if _id_re.match(id) is None:
        return redirect(url_for('user.index'))

    session = Session()
    user = session.query(User).filter_by(id=id).first()
    if not user is None:
        session.delete(user)
        session.commit()

    _recompute_authorized_keys()

    return redirect(url_for('user.index'))

@get("/<int:id>/recordings")
@authenticated
@templated("recordings.xhtml")
def recordings(id):
    session = Session()
    user = session.query(User).filter_by(id=id).first()
    if user is None:
        return redirect(url_for('user.index'))

    return dict(user=user)

@get("/<int:id>/recordings/json")
@authenticated
@jsonr()
def recordings_json(id):
    session_id = request.values.get("session", None)

    if session_id is None:
        return redirect(url_for('user.index'))

    vault = Vault()
    data = vault.retrieve(str(id)+"@"+session_id)
    if data is None:
        return redirect(url_for('user.index'))

    return data

@get("/<int:id>/recordings/ttyrec")
@authenticated
def recordings_ttyrec(id):
    session_id = request.values.get("session", None)
    if session_id is None:
        return redirect(url_for('user.index'))

    vault = Vault()
    data = vault.retrieve(str(id)+"@"+session_id)
    if data is None:
        return redirect(url_for('user.index'))

    data = json.loads(data)
    result = []
    for rec in data['recording']:
        if not "raw" in rec:
            continue
        secs = rec['delay'] // 1000000
        micro = rec['delay'] % 1000000
        raw = base64.b64decode(rec['raw'])
        result.append(struct.pack("<I", secs))
        result.append(struct.pack("<I", micro))
        result.append(struct.pack("<I", len(raw)))
        result.append(raw)

    response = make_response("".join(result))
    response.headers['Content-Type'] = "application/ttyrec"
    response.headers['Content-Disposition'] = \
        "attachment; filename=%s.ttyrec" % session_id
    return response
