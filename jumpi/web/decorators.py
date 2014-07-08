#-*- coding: utf-8 -*-

from flask import render_template, request, session, redirect, url_for, app
from functools import wraps

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


