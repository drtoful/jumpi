#-*- coding: utf-8 -*-

import json
import bcrypt
import base64
import os

from flask import render_template, request, session, redirect, url_for
from flask import make_response
from functools import wraps
from jumpi.api import APIAuth, APIStore

def authenticated(f):
    @wraps(f)
    def decorator(*args, **kwargs):
        def authenticate():
            return redirect(url_for('ui.login'))

        if "store_locked" in session:
            del session['store_locked']
        if session.get('bearer', None) is None:
            return authenticate()

        api = APIAuth()
        if not api.validate():
            del session['username']
            del session['bearer']
            return authenticate()

        api = APIStore()
        session['store_locked'] = api.status()

        return f(*args, **kwargs)
    return decorator

def templated(template):
    def decorator(f):
        @wraps(f)
        def template_function(*args, **kwargs):
            # error when no template is given
            if template is None:
                raise Exception("no template given")

            # get the context from the executed function
            context = None
            context = f(*args, **kwargs)

            if context is None:
                context = {}
            elif not isinstance(context, dict):
                return context

            context['session'] = session

            # render the context using given template
            response = render_template(template, **context)
            return response

        return template_function
    return decorator
