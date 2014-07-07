#-*- coding: utf-8 -*-

import requests

class Agent(object):
    def ping(self):
        try:
            req = requests.get("http://127.0.0.1:42000/ping")
            if req.status_code == 200:
                data = req.json()
                if data['pong']:
                    return (False, "Agent is locked, unlock first")
                return (True, None)
            return (False, "Agent response error")
        except requests.exceptions.ConnectionError:
            return (False, "Could not contact agent")

