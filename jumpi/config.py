#-*- coding: utf-8 -*-

import os
import ConfigParser

try:
    import pwd
    HOME_DIR = pwd.getpwuid(os.getuid()).pw_dir
except:
    HOME_DIR = os.path.expanduser("~")

class JumpiConfig(object):
    VAULT_ITERATIONS = 500
    VAULT_COMPLEXITY = 8
    CIPHER_ITERATIONS = 100
    AGENT_HOST = "127.0.0.1"
    AGENT_PORT = 42000

    def __init__(self):
        file = os.path.join(HOME_DIR, "jumpi-agent.cfg")

        self.parser = None
        if os.path.isfile(file):
            self.parser = ConfigParser.SafeConfigParser()
            self.parser.read(file)

    def getint(self, section, key, default=None):
        if self.parser is None:
            return default

        if self.parser.has_option(section, key):
            return self.parser.getint(section, key)
        return default

    def get(self, section, key, default=None):
        if self.parser is None:
            return default

        if self.parser.has_option(section, key):
            return self.parser.get(section, key)
        return default

_config = None
def get_config():
    global _config

    if _config is None:
        _config = JumpiConfig()
    return _config

