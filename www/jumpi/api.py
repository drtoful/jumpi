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

    def get(self, endpoint):
        uri = "%s%s" % (self.base_uri, endpoint)
        bearer = session.get("bearer", "_no_auth_")
        r = requests.get(uri, headers = {
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
