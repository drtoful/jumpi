#-*- coding: utf-8 -*-

import pyotp
import json
import datetime
import calendar

from jumpi.sh.agent import Vault

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

class TwoFactor(object):
    def __init__(self, user):
        self.user = user

    def setup(self, type="totp"):
        if not type in ["totp", "hotp"]:
            return False

        secret = pyotp.random_base32()
        otp = {
            'totp': pyotp.TOTP(secret),
            'hotp': pyotp.HOTP(secret)
        }.get(type)

        vault = Vault()
        req = vault.store(str(self.user.id)+"@otp", json.dumps({
            'type': type,
            'count': 1,
            'last': 0,
            'secret': secret
        }))
        if not req:
            return False

        print_qrcode(otp.provisioning_uri("jumpi/"+self.user.fullname))
        req = self.validate()
        if req:
            self.user.update("twofactor", True)

        return req

    def remove(self):
        return self.user.update("twofactor", False)

    def validate(self):
        import getpass

        def _now_timestamp():
            return calendar.timegm(round_time().timetuple())

        vault = Vault()
        req = vault.retrieve(str(self.user.id)+"@otp")
        if req is None:
            return False

        data = json.loads(req)
        def _validate_hotp(code):
            otp = pyotp.HOTP(data['secret'])
            return otp.verify(int(code), data['count'])

        def _validate_totp(code):
            otp = pyotp.TOTP(data['secret'])
            return otp.verify(int(code)) and _now_timestamp() > data['last']

        func = {
            'hotp': _validate_hotp,
            'totp': _validate_totp,
        }.get(data['type'], lambda x: False)

        if data['type'] == "hotp":
            print "Token Count: %d" % data['count']
        code = getpass.getpass("Token:")
        result = func(code)

        # need to update datastructure on success
        if result:
            data['count'] += 1
            data['last'] = _now_timestamp()

            req = vault.store(str(self.user.id)+"@otp", json.dumps(data))
            if not req:
                return False

        return result

