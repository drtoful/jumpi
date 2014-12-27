#-*- coding: utf-8 -*-

class Authenticator(object):
    def __init__(self):
        from jumpi.sh.twofactor.google import GoogleHOTPAuthenticator
        from jumpi.sh.twofactor.google import GoogleTOTPAuthenticator

        self.authenticators = [
            GoogleHOTPAuthenticator(),
            GoogleTOTPAuthenticator()
        ]

    def validate(self, user):
        pass

    def setup(self, user):
        if user.need_otp():
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
