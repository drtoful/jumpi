#-*- coding: utf-8 -*-

import json
import re

from flask import request, make_response
from functools import wraps

_date_re = re.compile("\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}")

def compose_json_response(status, **msg):
    content = json.dumps(msg)

    response = make_response(content)
    response.headers['Content-Type'] = "application/json; charset=utf-8"
    return response

def validate_json_field(data, type):
    if type == "integer":
        return isinstance(data, int)
    if type == "string":
        return isinstance(data, basestring)
    if type == "date":
        return isinstance(data, basestring) and _date_re.match(data)
    if type == "boolean":
        return isinstance(data, bool)
    return False

def json_required():
    def decorator(f):
        @wraps(f)
        def _decorator(*args, **kwargs):
            # according to documentation if request.json is None, then
            # content-type was not set to application/json or the data
            # could not be parsed as json
            if request.json is None:
                return compose_json_response(400, error="json_required",
                    error_long="JSON data and content-type required")

            return f(*args, **kwargs)

        return _decorator
    return decorator

def json_validate(required, **fields):
    def decorator(f):
        @wraps(f)
        def _decorator(*args, **kwargs):
            data = request.json
            set_required = set(required)
            set_fields = set(data.keys())

            # check if all required fields are set
            if not set_required.issubset(set_fields):
                missing = set_required.difference(
                    set_required.intersection(set_fields))
                return compose_json_response(400, error="json_missing_fields",
                    error_long="Missing required fields in JSON object: " +
                    ",".join(missing))

            # check all fields for correct types
            for field in fields.keys():
                if field in data.keys() and not \
                    validate_json_field(data[field], fields[field]):
                    return compose_json_response(400, error="json_wrong_type",
                        error_long="Incorrect type for field '"+
                        str(field) + "'; needs type '" + fields[field] + "'")

            return f(*args, **kwargs)

        return _decorator
    return decorator

