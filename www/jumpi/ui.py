#-*- coding: utf-8 -*-

import functools

from flask import Blueprint, redirect, url_for, request, session
from jumpi.decorators import templated, authenticated
from jumpi.api import APIAuth, APISecrets, APIStore

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
@authenticated
@templated("secrets.xhtml")
def secrets():
    api = APISecrets()
    return dict(secrets = api.list(0, 10))

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
