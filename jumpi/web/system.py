#-*- coding: utf-8 -*-

import functools
import SecureString

from flask import Blueprint, redirect, url_for, request, session
from jumpi.web.decorators import templated, authenticated
from jumpi.sh.agent import Vault
from jumpi.web.utils import WebPass

system = Blueprint("system", __name__)
get = functools.partial(system.route, methods=['GET'])
post = functools.partial(system.route, methods=['POST'])

@get("/")
@authenticated
@templated("system.xhtml")
def index():
    vault = Vault()
    return dict(
        agent = not vault.is_locked()
    )

@post("/unlock")
@authenticated
def unlock():
    vault = Vault()
    vault.unlock(request.form['passphrase'])
    SecureString.clearmem(request.form['passphrase'])

    return redirect(url_for('system.index'))

@post("/changepw")
@authenticated
def changepw():
    checker = WebPass()

    old = request.form['pw_old']
    new1 = request.form['pw_new1']
    new2 = request.form['pw_new2']

    if checker.verify(old) and new1 == new2:
        checker.update(new1)
        session["authenticated"] = False # force reauthentication

    return redirect(url_for('system.index'))
