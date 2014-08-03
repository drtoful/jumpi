#-*- coding: utf-8 -*-

import requests
import json
import os
import ConfigParser


class Agent(object):
    def __init__(self, host="127.0.0.1", port=42000):
        file = os.path.expanduser("~")
        file = os.path.join(file, "jumpi-agent.cfg")
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
        return store_data(username+"@"+hostname, key)

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
