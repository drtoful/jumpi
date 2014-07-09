#-*- coding: utf-8 -*-

import functools
import SecureString

from flask import Blueprint, redirect, url_for, request
from jumpi.web.decorators import templated
from jumpi.sh.agent import Agent

system = Blueprint("system", __name__)
get = functools.partial(system.route, methods=['GET'])
post = functools.partial(system.route, methods=['POST'])

@get("/")
@templated("system.xhtml")
def index():
    agent = Agent()
    return dict(
        agent = agent.ping()
    )

@post("/unlock")
def unlock():
    agent = Agent()

    agent.unlock(request.form['passphrase'])
    print request.form
    SecureString.clearmem(request.form['passphrase'])
    print request.form

    return redirect(url_for('system.index'))
