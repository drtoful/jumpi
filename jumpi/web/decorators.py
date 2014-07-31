#-*- coding: utf-8 -*-

import json
import bcrypt

from flask import render_template, request, session, redirect, url_for, app
from flask import make_response
from functools import wraps
from jumpi.sh.agent import Agent
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

        if not auth or not check_auth():
            return authenticate()

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
            agent = Agent()
            resp, _ = agent.ping()
            context['agent_unlock'] = resp

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
            if not isinstance(context, dict):
                return context

            response = make_response(json.dumps(context))
            response.headers['Content-Type'] = "application/json"

            return response


        return json_function
    return decorator
