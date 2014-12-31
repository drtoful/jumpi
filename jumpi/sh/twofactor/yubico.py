#-*- coding: utf-8 -*-

import getpass
import json

from jumpi.config import get_config
from jumpi.sh import log
from jumpi.sh.agent import Vault

class YubicoAuthenticator(object):
    def __init__(self):
        from yubico_client.yubico import DEFAULT_API_URLS

        config = get_config()
        self.cas = None
        if not config.get("yubico", "ca_path", None) is None:
            self.cas = config.get("yubico", "ca_path")

        self.urls = DEFAULT_API_URLS
        if not config.get("yubico", "api_url", None) is None:
            self.urls = config.get("yubico", "api_url").split(",")

    def validate(self, user):
        from yubico_client.yubico import Yubico
        from yubico_client.yubico_exceptions import YubicoError

        vault = Vault()
        req = vault.retrieve(str(user.id)+"@otp")
        if req is None:
            return False

        data = json.loads(req)
        otp = Yubico(data['device_id'], key=data['key'], api_urls=self.urls,
            ca_certs_bundle_path=self.cas)

        token = getpass.getpass("Token:")
        try:
            return otp.verify(token)
        except Exception as exc:
            log.error("yubico otp verification failed for user=%d: %s",
                user.id, exc.message)
        except YubicoError as exc:
            log.error("yubico otp verification failed for user=%d: %s",
                user.id, str(exc))

        return False


    def setup(self, user):
        from yubico_client.otp import OTP

        # ask for API key, or none
        key = getpass.getpass("API Key (or press enter for none):")
        if key == "":
            key = None

        # ask for token
        token = getpass.getpass("Token: ")
        otp = OTP(token, True)

        print "Setting up TwoFactor authentication " \
            "for YubiKey-ID '%s'" % otp.device_id

        vault = Vault()
        req = vault.store(str(user.id)+"@otp", json.dumps({
            'type': "yubico",
            'device_id': otp.device_id,
            'key': key
        }))
        if not req:
            return False

        req = self.validate(user)
        if req:
            log.info("user=%d has activated 2fa 'yubico', device_id=%s",
                user.id, otp.device_id)
            user.update("twofactor", True)

        return req

    @property
    def info(self):
        return ("yubico", "Yubico YubiKey OTP")
