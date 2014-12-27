#-*- coding: utf-8 -*-

import json
import datetime
import calendar

from jumpi.sh.agent import Vault
from jumpi.sh import log

def print_qrcode(value):
    """
        convert 'value' into qrcode and print it on the terminal. misuses
        terminal codes to draw it.

        adapted from nodejs (https://github.com/gtanner/qrcode-terminal)
    """
    import qrcode
    qr = qrcode.QRCode(border=1)
    qr.add_data(value)
    qr.make()

    black = "\033[40m  \033[0m"
    white = "\033[47m  \033[0m"

    for row in qr.get_matrix():
        print "".join([(white,black)[x] for x in row])

def round_time(dt=None, round=30, round_up=True):
    if dt is None: dt = datetime.datetime.now()
    seconds = (dt - dt.min).seconds
    rounding = (seconds+round/2) // round*round
    ndt = dt + datetime.timedelta(0, rounding-seconds, -dt.microsecond)
    if round_up and (ndt-dt).total_seconds() <= 0:
        return ndt + datetime.timedelta(seconds=round)
    return ndt

class GoogleHOTPAuthenticator(object):
    def validate(self, user):
        import getpass
        import pyotp

        vault = Vault()
        req = vault.retrieve(str(user.id)+"@otp")
        if req is None:
            return False

        data = json.loads(req)
        otp = pyotp.HOTP(data['secret'])

        print "Token Count: %d" % data['count']
        code = getpass.getpass("Token:")
        try:
            result = otp.verify(int(code), data['count'])
        except:
            result = False

        # need to update datastructure on success
        if result:
            data['count'] += 1

            req = vault.store(str(user.id)+"@otp", json.dumps(data))
            if not req:
                return False
        else:
            print "Invalid Token!"

        return result

    def setup(self, user):
        import pyotp
        secret = pyotp.random_base32()
        otp = pyotp.TOTP(secret)

        vault = Vault()
        req = vault.store(str(user.id)+"@otp", json.dumps({
            'type': "hotp",
            'count': 1,
            'secret': secret
        }))
        if not req:
            return False

        print_qrcode(otp.provisioning_uri("jumpi/"+user.fullname))
        req = self.validate(user)
        if req:
            log.info("user=%d has activated 2fa 'hotp'", user.id)
            user.update("twofactor", True)

        return req

    @property
    def info(self):
        return ("hotp", "HOTP (GoogleAuthenticator compatible)")

class GoogleTOTPAuthenticator(object):
    def validate(self, user):
        import getpass
        import pyotp

        def _now_timestamp():
            return calendar.timegm(round_time().timetuple())

        vault = Vault()
        req = vault.retrieve(str(user.id)+"@otp")
        if req is None:
            return False

        data = json.loads(req)
        otp = pyotp.TOTP(data['secret'])

        code = getpass.getpass("Token:")
        try:
            result = otp.verify(int(code)) and _now_timestamp() > data['last']
        except:
            result = False

        # need to update datastructure on success
        if result:
            data['last'] = _now_timestamp()

            req = vault.store(str(user.id)+"@otp", json.dumps(data))
            if not req:
                return False
        else:
            print "Invalid Token!"

        return result

    def setup(self, user):
        import pyotp

        secret = pyotp.random_base32()
        otp = pyotp.TOTP(secret)

        vault = Vault()
        req = vault.store(str(user.id)+"@otp", json.dumps({
            'type': "totp",
            'last': 0,
            'secret': secret
        }))
        if not req:
            return False

        print_qrcode(otp.provisioning_uri("jumpi/"+user.fullname))
        req = self.validate(user)
        if req:
            log.info("user=%d has activated 2fa 'totp'", user.id)
            user.update("twofactor", True)

        return req

    @property
    def info(self):
        return ("totp", "TOTP (GoogleAuthenticator compatible)")

