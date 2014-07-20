#-*- coding: utf-8 -*-

import functools
import SecureString
import re

from flask import Blueprint, redirect, url_for, request
from jumpi.web.decorators import templated, jsonr
from jumpi.web.user import _recompute_authorized_keys
from jumpi.db import Session, Tunnel, User, TunnelPermission
from jumpi.sh.agent import Agent

tunnel = Blueprint("tunnel", __name__)
get = functools.partial(tunnel.route, methods=['GET'])
post = functools.partial(tunnel.route, methods=['POST'])

_destination_re = re.compile("^[A-Z][A-Z0-9\.\-_]+$", re.IGNORECASE)
_ip_re = re.compile("[0-9\.]+")
_port_re = re.compile("[0-9]+")

@get("/")
@templated("tunnel.xhtml")
def index():
    session = Session()
    tunnels = session.query(Tunnel).order_by(Tunnel.id)
    users = session.query(User).all()

    return dict(tunnels = tunnels, users=users)

@get("/permissions")
@jsonr()
def get_permissions():
    try:
        session = Session()
        tunnel = session.query(Tunnel).filter_by(id=request.args['dbid']).first()

        permissions = [{'id': x.user.id, 'text': x.user.fullname}
            for x in tunnel.permissions]
        return dict(permissions=permissions)
    except:
        pass

    return dict()

@post("/permissions")
def save_permissions():
    dbid = request.form.get("dbid", None)
    if dbid is None:
        return redirect(url_for('tunnel.index'))

    permissions = request.form.getlist("perms[]")
    session = Session()

    perms = session.query(TunnelPermission).filter_by(tunnel_id=dbid)
    map(session.delete, perms)
    print permissions
    for id in permissions:
        perm = TunnelPermission(tunnel_id=dbid, user_id=id)
        _recompute_authorized_keys()
        session.add(perm)

    session.commit()
    return redirect(url_for('tunnel.index'))

@post("/delete")
def delete_tunnel():
    dbid = request.form.get("id", None)
    if dbid is None:
        return redirect(url_for('tunnel.index'))

    session = Session()
    tunnel = session.query(Tunnel).filter_by(id=dbid).first()
    if not tunnel is None:
        session.delete(tunnel)
        session.commit()

    return redirect(url_for('tunnel.index'))

@post("/add")
def add_tunnel():
    destination = request.form.get("destination", "")
    port = request.form.get("port", "")

    if (_destination_re.match(destination) is None and \
        _ip_re.match(destination) is None) or _port_re.match(port) is None:
        return redirect(url_for('tunnel.index'))

    tunnel = Tunnel(
        destination = destination,
        port = int(port)
    )

    session = Session()
    session.add(tunnel)
    session.commit()

    return redirect(url_for('tunnel.index'))

