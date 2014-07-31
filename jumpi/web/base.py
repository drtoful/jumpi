#-*- coding: utf-8 -*-

import functools
from flask import Blueprint, redirect, url_for
from jumpi.web.decorators import templated, authenticated

base = Blueprint("base", __name__)
get = functools.partial(base.route, methods=['GET'])
post = functools.partial(base.route, methods=['POST'])

@get("/")
@authenticated
@templated("base.xhtml")
def index():
    return dict()

