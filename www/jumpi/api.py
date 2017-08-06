#-*- coding: utf-8 -*-

import requests
import json

from flask import session
from jumpi.config import JumpiConfig

class API(object):
    def __init__(self):
        config = JumpiConfig()
        self.base_uri = "http://%s:%d/api" % (config.JUMPI_HOST, \
            config.JUMPI_PORT)

    def _parse_response(self, r):
        if r is None:
            return False, None
        try:
            data = r.json()
            if r.status_code == 500:
                print "server error from API server: "+data.get('response', "")
            return r.status_code == 200, data.get('response', None)
        except:
            pass
        return False, None

    def get(self, endpoint, data = None):
        uri = "%s%s" % (self.base_uri, endpoint)
        bearer = session.get("bearer", "_no_auth_")
        r = requests.get(uri, params = data, headers = {
            'Authorization': "Bearer %s" % bearer})
        return self._parse_response(r)

    def post(self, endpoint, data):
        uri = "%s%s" % (self.base_uri, endpoint)
        bearer = session.get("bearer", "_no_auth_")
        try:
            r = requests.post(uri, data = json.dumps(data), headers = {
                'content-type' : "application/json",
                'Authorization': "Bearer %s" % bearer})
            return self._parse_response(r)
        except:
            pass
        return False, None

    def delete(self, endpoint, data = None):
        uri = "%s%s" % (self.base_uri, endpoint)
        bearer = session.get("bearer", "_no_auth_")
        try:
            r = requests.delete(uri, params = data, headers = {
                'Authorization': "Bearer %s" % bearer})
            return self._parse_response(r)
        except:
            pass
        return False, None

api = API()

class APIAuth(object):
    def login(self, username, password):
        ok, session = api.post("/v1/auth/login", dict( \
            username = username, password = password))
        if ok:
            return session
        return None

    def validate(self):
        ok, _ = api.get("/v1/auth/validate")
        return ok

    def logout(self):
        ok, _ = api.get("/v1/auth/logout")
        return ok

class APIStore(object):
    def is_locked(self):
        ok, val = api.get("/v1/store/status")
        if ok:
            return val.get("locked", True)
        return True

    def unlock(self, password):
        ok, _ = api.post("/v1/store/unlock", dict( \
            password = password))
        return ok

    def lock(self):
        ok, _ = api.post("/v1/store/lock", None)
        return ok

class APISecrets(object):
    def list(self, skip, limit):
        ok, vals = api.get("/v1/secrets/list", dict( \
            skip = skip, limit = limit))
        if ok:
            return vals
        return []

    def set(self, name, type, data):
        ok, err = api.post("/v1/secrets", dict( \
            id = name, type = type, data = data))
        if not ok:
            return err
        return None

    def delete(self, id):
        ok, _ = api.delete("/v1/secrets/"+id)
        return ok

class APITargets(object):
    def list(self, skip, limit):
        ok, vals = api.get("/v1/targets/list", dict( \
            skip = skip, limit = limit))

        if ok:
            return vals
        return []

    def set(self, user, hostname, port, secret):
        ok, err = api.post("/v1/targets", dict( \
            user = user, host = hostname, \
            port = port, secret = secret))
        if not ok:
            return err
        return None

    def delete(self, id):
        ok, _ = api.delete("/v1/targets/"+id)
        return ok

class APIUsers(object):
    def list(self, skip, limit):
        ok, vals = api.get("/v1/users/list", dict( \
            skip = skip, limit = limit))

        if ok:
            return vals
        return []

    def set(self, name, pub):
        ok, err = api.post("/v1/users", dict( \
            name = name, pub = pub))
        if not ok:
            return err
        return None

    def delete(self, id):
        ok, _ = api.delete("/v1/users/"+id)
        return ok

class APIRoles(object):
    def list(self, skip, limit):
        ok, vals = api.get("/v1/roles/list", dict( \
            skip = skip, limit = limit))

        if ok:
            return vals
        return []

    def set(self, name, rex_user, rex_target, require_2fa):
        ok, err = api.post("/v1/roles", dict( \
            name = name, rex_user = rex_user, rex_target = rex_target, \
            require_2fa = require_2fa))
        if not ok:
            return err
        return None

    def delete(self, id):
        ok, _ = api.delete("/v1/roles/"+id)
        return ok

class APICasts(object):
    def list(self, skip, limit):
        ok, vals = api.get("/v1/casts/list", dict( \
            skip = skip, limit = limit))

        if ok:
            return vals
        return []

    def get(self, id):
        ok, data = api.get("/v1/casts/"+id)
        if not ok:
            return None
        return data
