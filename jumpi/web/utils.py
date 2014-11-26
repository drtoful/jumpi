#-*- coding: utf-8 -*-

import os
import bcrypt

from pyvault.utils import constant_time_compare
from jumpi.config import HOME_DIR

class WebPass(object):
    def __init__(self):
        self.filename = os.path.join(HOME_DIR, "jumpi-web.pass")
        if not os.path.isfile(self.filename):
            self.update("admin")
        else:
            with open(self.filename, "r") as fp:
                self.hash = (fp.read()).strip().encode('utf-8')

    def verify(self, password):
        return constant_time_compare(bcrypt.hashpw(
            password.encode('utf-8'), self.hash),
            self.hash)


    def update(self, password):
        self.hash = bcrypt.hashpw(password.encode('utf-8'),
            bcrypt.gensalt(12)).encode('utf-8')

        with open(self.filename, "w") as fp:
            print >>fp, self.hash

