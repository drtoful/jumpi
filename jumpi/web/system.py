#-*- coding: utf-8 -*-

import functools
from flask import Blueprint, redirect, url_for
from jumpi.web.decorators import templated

system = Blueprint("system", __name__)
get = functools.partial(system.route, methods=['GET'])
post = functools.partial(system.route, methods=['POST'])

@get("/")
@templated("system.xhtml")
def index():
    return dict()

