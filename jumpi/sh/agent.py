#-*- coding: utf-8 -*-

import json
import datetime
from jumpi.config import get_config, JumpiConfig

def format_datetime(value):
    return value.strftime("%Y-%m-%d %H:%M:%S")

class Agent(object):
    def __init__(self):
        config = get_config()
        host = config.get("agent", "host", JumpiConfig.AGENT_HOST)
        port = config.getint("agent", "port", JumpiConfig.AGENT_PORT)

        self.url = "http://%s:%d" % (host, port)

    def _doit(self, method, endpoint, **arguments):
        import requests

        func = {
            "GET": requests.get,
            "POST": requests.post,
            "PUT": requests.put,
            "PATCH": requests.patch,
            "DELETE": requests.delete
        }.get(method, requests.get)

        try:
            req = func("%s%s" % (self.url, endpoint),
                data = json.dumps(arguments),
                headers = {'content-type': "application/json; charset=utf-8"})
            if req.status_code == 200:
                try:
                    return req.json()
                except ValueError:
                    return req.text
        except requests.exceptions.ConnectionError:
            pass

        return None

    def get(self, endpoint, **arguments):
        return self._doit("GET", endpoint, **arguments)

    def post(self, endpoint, **arguments):
        return self._doit("POST", endpoint, **arguments)

    def put(self, endpoint, **arguments):
        return self._doit("PUT", endpoint, **arguments)

    def patch(self, endpoint, **arguments):
        return self._doit("PATCH", endpoint, **arguments)

    def delete(self, endpoint, **arguments):
        return self._doit("DELETE", endpoint, **arguments)

class Target(object):
    def __init__(self, id):
        self.id = id
        self._data = None

    def load(self):
        if not self._data is None:
            return

        agent = Agent()
        req = agent.get("/target", id=self.id)
        self._data = req

    @property
    def port(self):
        return self._data.get('port', 22)

    @property
    def type(self):
        return self._data.get('type', "password")

class File(object):
    def __init__(self, filename):
        self.filename = filename
        self._data = None

    @classmethod
    def save(self, **data):
        agent = Agent()
        req = agent.put("/file", **data)
        return not req is None

    @property
    def basename(self):
        return self._data.get('basename', None)

    @property
    def created(self):
        return self._data.get('created', None)

    @property
    def size(self):
        return self._data.get('size', None)

    def delete(self):
        agent = Agent()
        req = agent.delete("/file", filename=self.filename)
        return not req is None

class Permission(object):
    def __init__(self, id):
        self.id = id
        self._data = None

    @property
    def target(self):
        result = Target(self.target_id)
        result.load()
        return result

    @property
    def target_id(self):
        return self._data.get('target_id', None)

    @property
    def user_id(self):
        return self._data.get('user_id', None)

class User(object):
    def __init__(self, id):
        try:
            self.id = int(id)
        except ValueError:
            self.id = -1
        self.load()

    def load(self):
        self._load_info()
        self._load_files()
        self._load_permissions()

    def is_valid(self):
        return not self._info is None

    def _load_info(self):
        agent = Agent()
        self._info = agent.get("/user/info", user=self.id)

    def _load_permissions(self):
        agent = Agent()
        req = agent.get("/user/permissions", user=self.id)

        self._permissions = []
        if not req is None:
            def _perm(data):
                result = Permission(data.get('id', 0))
                result._data = data
                return result

            self._permissions = [_perm(x) for x in req['permissions']]

    def _load_files(self):
        agent = Agent()
        req = agent.get("/user/files", user=self.id)

        self._files = []
        if not req is None:
            def _file(data):
                result = File(data.get('filename', ""))
                result._data = data
                return result

            self._files = [_file(x) for x in req['files']]

    @property
    def fullname(self):
        return self._info.get('fullname', None)

    @property
    def target_permissions(self):
        return self._permissions

    @property
    def files(self):
        return self._files

    def update(self, key, value):
        agent = Agent()
        req = agent.patch("/user/info", **dict(
            [("user", self.id), (key,value)]))
        return not req is None

    def add_recording(self, **data):
        agent = Agent()
        req = agent.put("/recording", **data)
        return not req is None

class Vault(object):
    def __init__(self):
        self.agent = Agent()

    def is_locked(self):
        status = self.agent.get("/vault/status")
        if not status is None:
            return status.get('locked', True)
        return True

    def unlock(self, passphrase):
        req = self.agent.post("/vault/unlock", passphrase=passphrase)
        if not req is None:
            return True
        return False

    def store(self, key, value):
        req = self.agent.put("/vault/store", key=key, value=value)
        if not req is None:
            return True
        return False

    def retrieve(self, key):
        req = self.agent.get("/vault/retrieve", key=key)
        if req is None:
            return req
        return req['value']

