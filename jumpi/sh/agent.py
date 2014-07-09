#-*- coding: utf-8 -*-

import requests
import json

class Agent(object):
    def __init__(self, host="127.0.0.1", port=42000):
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
