#-*- coding: utf-8 -*-

import json
import bcrypt
import base64
import os

from flask import render_template, request, session, redirect, url_for, app
from flask import make_response
from functools import wraps
from jumpi.sh.agent import Vault
from jumpi.web.utils import WebPass

def authenticated(f):
    @wraps(f)
    def decorator(*args, **kwargs):
        auth = request.authorization
        def check_auth():
            checker = WebPass()
            return auth.username == "admin" and checker.verify(auth.password)

        def authenticate():
            response = make_response('Authentication required!', 401)
            response.headers['WWW-Authenticate'] = "Basic realm=\"JumPi\""
            return response

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

            # status of agent
            vault = Vault()
            context['agent_unlock'] = not vault.is_locked()

            # render the context using given template
            response = render_template(template, **context)
            return response

        return template_function
    return decorator

def jsonr():
    def decorator(f):
        @wraps(f)
        def json_function(*args, **kwargs):
            context = f(*args, **kwargs)

            if not isinstance(context, dict) and \
                not isinstance(context, basestring):
                return context

            content = ""
            if isinstance(context, basestring):
                content = str(context)
            else:
                content = json.dumps(context)

            response = make_response(content)
            response.headers['Content-Type'] = "application/json"

            return response


        return json_function
    return decorator
