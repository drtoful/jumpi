#-*- coding: utf-8 -*-

import os
import ConfigParser

try:
    import pwd
    HOME_DIR = pwd.getpwuid(os.getuid()).pw_dir
except:
    HOME_DIR = os.path.expanduser("~")

class JumpiConfig(object):
    def __init__(self):
        file = os.path.join(HOME_DIR, "jumpi-agent.cfg")
        if os.path.isfile(file):
            self.parser = ConfigParser.SafeConfigParser()
            self.parser.read(file)

    def getint(self, section, key, default=None):
        if self.parser.has_option(section, key):
            return self.parser.getint(section, key)
        return default

    def get(self, section, key, default=None):
        if self.parser.has_option(section, key):
            return self.parser.get(section, key)
        return default

_config = None
def get_config():
    global _config

    if _config is None:
        _config = JumpiConfig()
    return _config

