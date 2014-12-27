#-*- coding: utf-8 -*-

import json

from jumpi.sh.agent import Vault
from jumpi.sh import log

class Authenticator(object):
    def __init__(self):
        from jumpi.sh.twofactor.google import GoogleHOTPAuthenticator
        from jumpi.sh.twofactor.google import GoogleTOTPAuthenticator

        self.authenticators = [
            GoogleHOTPAuthenticator(),
            GoogleTOTPAuthenticator()
        ]

    def validate(self, user):
        if not user.need_otp():
            return True

        vault = Vault()
        otp = vault.retrieve(str(user.id)+"@otp")
        if otp is None:
            log.error("user=%d has corrupt otp. " \
                "no otp secret stored in vault", user.id)
            print "Corrupt OTP setup. Please contact admin!"
            return False

        typus = json.loads(otp)["type"]
        for i in xrange(0, len(self.authenticators)):
            auth = self.authenticators[i].info[0]
            if auth == typus:
                return self.authenticators[i].validate(user)

        log.error("user=%d has unknown otp type '%s'", user.id, typus)
        print "Corrupt OTP setup. Unknown OTP type. Please contact admin!"
        return False

    def setup(self, user):
        if user.need_otp():
            print "TwoFactor Authentication already setup!"
            return False

        print "Choose Authenticator type:"
        for i in xrange(0, len(self.authenticators)):
            print "%d) %s" % (i, self.authenticators[i].info[1])
        print ""
        print "c) cancel"
        print ""

        choice = raw_input("Your choice? ")
        if choice == "c":
            return False

        try:
            choice = int(choice)
            if choice < 0 or choice >= len(self.authenticators):
                raise Exception()
        except:
            print "Invalid choice!"
            return False

        auth = self.authenticators[choice]
        return auth.setup(user)

authenticator = Authenticator()
