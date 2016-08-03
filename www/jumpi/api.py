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
            if r.status_code == 200:
                return True, data['response']
            else:
                return False, data['description']
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

    def delete(self, endpoint):
        uri = "%s%s" % (self.base_uri, endpoint)
        bearer = session.get("bearer", "_no_auth_")
        try:
            r = requests.delete(uri, headers = {
                'Authorization': "Bearer %s" % bearer})
            return self._parse_response(r)
        except:
            pass
        return False, None

api = API()

class APIAuth(object):
    def login(self, username, password):
        ok, session = api.post("/auth/login", dict( \
            username = username, password = password))
        if ok:
            return session
        return None

    def validate(self):
        ok, _ = api.get("/auth/validate")
        return ok

    def logout(self):
        ok, _ = api.get("/auth/logout")
        return ok

class APIStore(object):
    def is_locked(self):
        ok, val = api.get("/store/status")
        if ok:
            return val
        return True

    def unlock(self, password):
        ok, _ = api.post("/store/unlock", dict( \
            password = password))
        return ok

    def lock(self):
        ok, _ = api.post("/store/lock", None)
        return ok

class APISecrets(object):
    def list(self, skip, limit):
        ok, vals = api.get("/secrets", dict( \
            skip = skip, limit = limit))

        if ok:
            nvals = []
            for k in vals:
                try:
                    k["value"] = json.loads(k["value"])
                    del k["value"]["data"]
                    nvals = nvals + [k]
                except:
                    pass

            return nvals
        return None

    def set(self, name, type, data):
        ok, err = api.post("/secrets", dict( \
            id = name, type = type, data = data))
        if not ok:
            return err
        return None

    def delete(self, id):
        ok, _ = api.delete("/secrets/%s" % id)
        return ok

class APITargets(object):
    def list(self, skip, limit):
        ok, vals = api.get("/targets", dict( \
            skip = skip, limit = limit))

        if ok:
            nvals = []
            for k in vals:
                try:
                    k["value"] = json.loads(k["value"])
                    nvals = nvals + [k]
                except:
                    pass

            return nvals
        return None

    def set(self, user, hostname, port, secret):
        ok, err = api.post("/targets", dict( \
            username = user, hostname = hostname, \
            port = port, secret = secret))
        if not ok:
            return err
        return None

    def delete(self, id):
        ok, _ = api.delete("/targets/%s" % id)
        return ok

class APIUsers(object):
    def list(self, skip, limit):
        ok, vals = api.get("/users", dict( \
            skip = skip, limit = limit))

        if ok:
            nvals = []
            for k in vals:
                try:
                    k["value"] = json.loads(k["value"])
                    nvals = nvals + [k]
                except:
                    pass

            return nvals
        return None

    def set(self, name, pub):
        ok, err = api.post("/users", dict( \
            name = name, pub = pub))
        if not ok:
            return err
        return None

    def delete(self, id):
        ok, _ = api.delete("/users/%s" % id)
        return ok
