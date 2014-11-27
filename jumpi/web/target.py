#-*- coding: utf-8 -*-

import functools
import SecureString
import re

from flask import Blueprint, redirect, url_for, request
from jumpi.web.decorators import templated, jsonr, authenticated
from jumpi.db import Session, Target, User, TargetPermission
from jumpi.sh.agent import Vault

target = Blueprint("target", __name__)
get = functools.partial(target.route, methods=['GET'])
post = functools.partial(target.route, methods=['POST'])

_username_re = re.compile("^[a-z][a-z0-9_\-]+", re.IGNORECASE)
_hostname_re = re.compile(
    "^(?=.{1,255}$)[0-9A-Za-z](?:(?:[0-9A-Za-z]|-)"+
    "{0,61}[0-9A-Za-z])?(?:\.[0-9A-Za-z](?:(?:[0-9A-Za-z]|-)"+
    "{0,61}[0-9A-Za-z])?)*\.?$",
    re.IGNORECASE
)
_ip_re = re.compile("[0-9\.]+")
_port_re = re.compile("[0-9]+")

@get("/")
@authenticated
@templated("target.xhtml")
def index():
    session = Session()
    targets = session.query(Target).order_by(Target.id)
    users = session.query(User).all()

    return dict(targets = targets, users=users)

@get("/permissions")
@authenticated
@jsonr()
def get_permissions():
    try:
        session = Session()
        target = session.query(Target).filter_by(id=request.args['dbid']).first()


        permissions = [{'id': x.user.id, 'text': x.user.fullname}
            for x in target.permissions]
        return dict(permissions=permissions)
    except:
        print "error occured!"

    return dict()

@post("/permissions")
@authenticated
def save_permissions():
    dbid = request.form.get("dbid", None)
    if dbid is None:
        return redirect(url_for('target.index'))

    permissions = request.form.getlist("perms[]")
    session = Session()

    perms = session.query(TargetPermission).filter_by(target_id=dbid)
    map(session.delete, perms)
    print permissions
    for id in permissions:
        perm = TargetPermission(target_id=dbid, user_id=id)
        session.add(perm)

    session.commit()
    return redirect(url_for('target.index'))

@post("/delete")
@authenticated
def delete_target():
    dbid = request.form.get("id", None)
    if dbid is None:
        return redirect(url_for('target.index'))

    session = Session()
    target = session.query(Target).filter_by(id=dbid).first()
    if not target is None:
        session.delete(target)
        session.commit()

    return redirect(url_for('target.index'))

@post("/add")
@authenticated
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

    vault = Vault()
    if vault.store(username+"@"+hostname, key):
        session = Session()
        session.merge(target)
        session.commit()

    return redirect(url_for('target.index'))

