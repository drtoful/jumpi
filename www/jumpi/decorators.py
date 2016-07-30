#-*- coding: utf-8 -*-

import json
import bcrypt
import base64
import os

from flask import render_template, request, session, redirect, url_for
from flask import make_response
from functools import wraps

def authenticated(f):
    @wraps(f)
    def decorator(*args, **kwargs):
        auth = request.authorization
        def check_auth():
            return True

        def authenticate():
            return redirect(url_for('ui.login'))

        if not session.get('authenticated', None) and \
            (not auth or not check_auth()):
            return authenticate()

        session['authenticated'] = True
        session['username'] = auth['username']
        if session.get('salt', None) is None:
            session['salt'] = base64.b64encode(os.urandom(12))

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
