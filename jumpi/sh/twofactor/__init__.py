#-*- coding: utf-8 -*-

import json

from jumpi.sh.agent import Vault
from jumpi.sh import log

class Authenticator(object):
    def __init__(self):
        self.authenticators = []

        try:
            from jumpi.sh.twofactor.google import GoogleHOTPAuthenticator
            from jumpi.sh.twofactor.google import GoogleTOTPAuthenticator

            self.authenticators += [
                GoogleHOTPAuthenticator(),
                GoogleTOTPAuthenticator(),
            ]
        except ImportError:
            pass

        try:
            from jumpi.sh.twofactor.yubico import YubicoAuthenticator

            self.authenticators += [
                YubicoAuthenticator()
            ]
        except ImportError:
            pass


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
            print "%d) %s" % (i+1, self.authenticators[i].info[1])
        print ""
        print "c) cancel"
        print ""

        choice = raw_input("Your choice? ")
        if choice == "c":
            return False

        try:
            choice = int(choice)-1
            if choice < 0 or choice >= len(self.authenticators):
                raise Exception()
        except:
            print "Invalid choice!"
            return False

        auth = self.authenticators[choice]
        try:
            return auth.setup(user)
        except ImportError:
            print "Authenticator not available! Please ask administrator for" + \
                "further information"
            log.error("user=%d tried to setup authenticator '%s' which " + \
                "resulted in import error; maybe missing dependencies?",
                user.id, self.authenticators[choice].info[0])

authenticator = Authenticator()
