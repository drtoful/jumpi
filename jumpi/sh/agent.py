#-*- coding: utf-8 -*-

import requests
import json
import os
import ConfigParser

from jumpi.sh import HOME_DIR

class User(object):
    class Target(object):
        def __init__(self, data):
            self._data = data

        @property
        def target_id(self):
            return self._data.get('target_id', None)

    class File(object):
        def __init__(self, data):
            self._data = data

        @property
        def filename(self):
            return self._data.get('filename', None)

        @property
        def basename(self):
            return self._data.get('filename', None)

        @property
        def created(self):
            return self._data.get('created', None)

        @property
        def size(self):
            return self._data.get('size', None)

    def __init__(self, agent, id):
        self.agent = agent
        self._id = id

        self._info = None
        self._targets = None
        self._files = None

        self.refresh()

    def refresh(self):
        self._load_info()
        if not self._targets is None:
            self._targets = None
            self._load_targets()
        if not self._files is None:
            self._files = None
            self._load_files()

    def is_valid(self):
        return not self._info is None

    def _load_info(self):
        self._info = self.agent.user_info(self._id)
        if not self._info is None:
            self._info = json.loads(self._info)

    def _load_targets(self):
        if self._targets is None:
            self._targets = self.agent.user_targets(self._id)
            if not self._targets is None:
                self._targets = json.loads(self._targets)
                self._targets = [User.Target(x) for x in self._targets]

    def _load_files(self):
        if self._files is None:
            self._files = self.agent.user_files(self._id)
            if not self._files is None:
                self._files = json.loads(self._files)
                self._files = [User.File(x) for x in self._files]

    @property
    def id(self):
        return self._info.get('id', None)

    @property
    def fullname(self):
        return self._info.get('fullname', None)

    @property
    def target_permissions(self):
        self._load_targets()
        return self._targets

    @property
    def files(self):
        self._load_files()
        return self._files

class Agent(object):
    def __init__(self, host="127.0.0.1", port=42000):
        file = os.path.join(HOME_DIR, "jumpi-agent.cfg")
        if os.path.isfile(file):
            parser = ConfigParser.SafeConfigParser()
            parser.read(file)

            if parser.has_option("agent", "host"):
                host = parser.get("agent", "host")
            if parser.has_option("agent", "port"):
                port = parser.getint("agent", "port")

        self.url = "http://%s:%d" % (host, port)

    def ping(self):
        try:
            req = requests.get("%s/ping" % self.url)
            if req.status_code == 200:
                data = req.json()
                if data['pong']:
                    return (False, "Agent is locked, unlock first")
                return (True, None)
            return (False, "Agent response error")
        except requests.exceptions.ConnectionError:
            return (False, "Could not contact agent")

    def unlock(self, passphrase):
        try:
            req = requests.post("%s/unlock" % self.url,
                data = json.dumps({'passphrase': passphrase}),
                headers = {'content-type': "application/json; charset=utf-8"})
            if req.status_code == 200:
                return True
            return False
        except requests.exceptions.ConnectionError:
            return False

    def store_data(self, id, data):
        try:
            req = requests.put("%s/store" % self.url,
                data = json.dumps({'id': id, 'key': data}),
                headers = {'content-type': "application/json; charset=utf-8"})
            if req.status_code == 200:
                return True
            return False
        except requests.exceptions.ConnectionError:
            return False

    def store(self, username, hostname, key):
        return self.store_data(username+"@"+hostname, key)

    def retrieve(self, id):
        try:
            req = requests.get("%s/retrieve" % self.url,
                data = json.dumps({'id': id}),
                headers = {'content-type': "application/json; charset=utf-8"})
            if req.status_code == 200:
                return req.text
            return None
        except requests.exceptions.ConnectionError:
            return None

    def user_info(self, id):
        try:
            req = requests.get("%s/user/%d/info" % (self.url, int(id)))
            if req.status_code == 200:
                return req.text
            return None
        except requests.exceptions.ConnectionError:
            return None

    def user_targets(self, id):
        try:
            req = requests.get("%s/user/%d/targets" % (self.url, int(id)))
            if req.status_code == 200:
                return req.text
            return None
        except requests.exceptions.ConnectionError:
            return None

    def user_files(self, id):
        try:
            req = requests.get("%s/user/%d/files" % (self.url, int(id)))
            if req.status_code == 200:
                return req.text
            return None
        except requests.exceptions.ConnectionError:
            return None

