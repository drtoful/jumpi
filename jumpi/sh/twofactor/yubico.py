#-*- coding: utf-8 -*-

import getpass
import json

from jumpi.config import get_config
from jumpi.sh import log
from jumpi.sh.agent import Vault
from yubico_client.yubico import Yubico, DEFAULT_API_URLS
from yubico_client.yubico_exceptions import YubicoError

class YubicoAuthenticator(object):
    def __init__(self):
        config = get_config()

        cas = None
        if not config.get("yubico", "ca_path", None) is None:
            cas = config.get("yubico", "ca_path")

        api_urls = DEFAULT_API_URLS
        if not config.get("yubico", "api_url", None) is None:
            api_urls = config.get("yubico", "api_url").split(",")

        api_clientid = ""
        if not config.get("yubico", "api_clientid", None) is None:
            api_clientid = config.get("yubico", "api_clientid")

        api_secret = None
        if not config.get("yubico", "api_secret", None) is None:
            api_secret = config.get("yubico", "api_secret")

        self.yubico = Yubico(api_clientid, key=api_secret, api_urls=api_urls,
            ca_certs_bundle_path=cas)

    def validate(self, user):
        vault = Vault()
        req = vault.retrieve(str(user.id)+"@otp")
        if req is None:
            return False

        data = json.loads(req)
        token = getpass.getpass("Token:")
        try:
            if not token[:12] == data['device_id']:
                return False
            return self.yubico.verify(token)
        except Exception as exc:
            log.error("yubico otp verification failed for user=%d: %s",
                user.id, exc.message)
        except YubicoError as exc:
            log.error("yubico otp verification failed for user=%d: %s",
                user.id, str(exc))

        return False


    def setup(self, user):
        # ask for token
        token = getpass.getpass("Token: ")
        print "Setting up TwoFactor authentication " \
            "for YubiKey-ID '%s'" % token[:12]

        vault = Vault()
        req = vault.store(str(user.id)+"@otp", json.dumps({
            'type': "yubico",
            'device_id': token[:12]
        }))
        if not req:
            return False

        req = self.validate(user)
        if req:
            log.info("user=%d has activated 2fa 'yubico', device_id=%s",
                user.id, token[:12])
            user.update("twofactor", True)

        return req

    @property
    def info(self):
        return ("yubico", "Yubico YubiKey OTP")
