#-*- coding: utf-8 -*-

import functools

from flask import Blueprint, redirect, url_for, request, session
from jumpi.decorators import templated, authenticated
from jumpi.api import APIAuth, APISecrets, APIStore
from jumpi.api import APITargets, APIUsers, APIRoles

uibp = Blueprint("ui", __name__)
get = functools.partial(uibp.route, methods=['GET'])
post = functools.partial(uibp.route, methods=['POST'])

@get("/")
@authenticated
@templated("base.xhtml")
def index():
    return dict()

@get("/login")
@post("/login")
@templated("login.xhtml")
def login():
    if request.method == "POST":
        username = request.form.get("username", "")
        password = request.form.get("password", "")

        api = APIAuth()
        sess = api.login(username, password)
        if sess is None:
            return dict(error = "Invalid Username/Password")

        session['username'] = username
        session['bearer'] = sess
        return redirect(url_for("ui.index"))

    return dict()

@get("/logout")
def logout():
    api = APIAuth()
    if api.logout():
        del session['username']
        del session['bearer']
    return redirect(url_for("ui.index"))

@get("/secrets")
@post("/secrets")
@authenticated
@templated("secrets.xhtml")
def secrets():
    api = APISecrets()
    error = None

    if request.method == "POST":
        type = 0
        try:
            type = int(request.form.get("type", 0))
        except:
            pass
        name = request.form.get("name", "")
        data = request.form.get("data", "")

        err = api.set(name, type, data)
        if not err is None:
            error = err

    page = 0
    try:
        page = int(request.args.get("p", 0))
    except:
        pass
    return dict(secrets = api.list(page*10, 10), error = error, page = page)

@get("/secrets/delete")
@authenticated
def delete_secret():
    id = request.args.get("id", None)
    if not id is None:
        api = APISecrets()
        api.delete(id)
    return redirect(url_for("ui.secrets"))

@get("/store")
@post("/store")
@authenticated
@templated("store.xhtml")
def store():
    api = APIStore()
    if request.method == "POST":
        action = request.form.get("action", "")
        if action == "unlock":
            password = request.form.get("password", "")
            api.unlock(password)
        if action == "lock":
            api.lock()

        return redirect(url_for("ui.store"))

    return dict()

@get("/targets")
@post("/targets")
@authenticated
@templated("targets.xhtml")
def targets():
    api = APITargets()
    error = None

    if request.method == "POST":
        user = request.form.get("user", "")
        host = request.form.get("host", "")
        port = 0
        try:
            port = int(request.form.get("port", 0))
        except:
            pass
        secret = request.form.get("secret", "")

        err = api.set(user, host, port, secret)
        if not err is None:
            error = err


    page = 0
    try:
        page = int(request.args.get("p", 0))
    except:
        pass

    return dict(targets = api.list(page*10, 10), page = page, error = error)

@get("/targets/delete")
@authenticated
def delete_target():
    id = request.args.get("id", None)
    if not id is None:
        api = APITargets()
        api.delete(id)
    return redirect(url_for("ui.targets"))

@get("/users")
@post("/users")
@authenticated
@templated("users.xhtml")
def users():
    api = APIUsers()
    error = None

    if request.method == "POST":
        name = request.form.get("name", "")
        pub = request.form.get("public", "")

        err = api.set(name, pub)
        if not err is None:
            error = err

    page = 0
    try:
        page = int(request.args.get("p", 0))
    except:
        pass

    return dict(users = api.list(page*10, 10), page = page, error = error)

@get("/users/delete")
@authenticated
def delete_user():
    id = request.args.get("id", None)
    if not id is None:
        api = APIUsers()
        api.delete(id)
    return redirect(url_for("ui.users"))

@get("/roles")
@post("/roles")
@authenticated
@templated("roles.xhtml")
def roles():
    api = APIRoles()
    error = None

    if request.method == "POST":
        name = request.form.get("name", "")
        rex_user = request.form.get("rex_user", "")
        rex_target = request.form.get("rex_target", "")

        err = api.set(name, rex_user, rex_target)
        if not err is None:
            error = err

    page = 0
    try:
        page = int(request.args.get("p", 0))
    except:
        pass

    return dict(roles = api.list(page*10, 10), page = page, error = error)

@get("/roles/delete")
@authenticated
def delete_role():
    id = request.args.get("id", None)
    if not id is None:
        api = APIRoles()
        api.delete(id)
    return redirect(url_for("ui.roles"))
