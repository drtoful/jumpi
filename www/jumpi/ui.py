#-*- coding: utf-8 -*-

import functools
from flask import Blueprint
from jumpi.decorators import templated, authenticated

uibp = Blueprint("ui", __name__)
get = functools.partial(uibp.route, methods=['GET'])
post = functools.partial(uibp.route, methods=['POST'])

@get("/")
@authenticated
@templated("base.xhtml")
def index():
    return dict()

@get("/login")
@templated("login.xhtml")
def login():
    return dict()
