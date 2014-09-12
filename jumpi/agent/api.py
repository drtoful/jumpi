#-*- coding: utf:8 -*-

import json
import os
import ConfigParser

from flask import Flask, request, Response
from pyvault import PyVault
from pyvault.backends.ptree import PyVaultPairtreeBackend
from pyvault.ciphers.aes import PyVaultCipherAES
from pyvault.ciphers import cipher_manager
from jumpi.agent import log, get_session_id, HOME_DIR

app = Flask(__name__)

class _JumpiAES(PyVaultCipherAES):
    def __init__(self):
        PyVaultCipherAES.__init__(self)

        file = os.path.join(HOME_DIR, "jumpi-agent.cfg")
        if os.path.isfile(file):
            parser = ConfigParser.SafeConfigParser()
            parser.read(file)

            if parser.has_option("cipher", "iterations"):
                self.KEYDERIV_ITERATIONS = parser.getint(
                    "cipher", "iterations")

    @property
    def id(self):
        return "aes-jumpi"

_backend = PyVaultPairtreeBackend(
    os.path.join(HOME_DIR, ".store")
)
_vault = PyVault(_backend)
cipher_manager.register("aes-jumpi", _JumpiAES())

@app.route("/unlock", methods=['POST'])
def unlock():
    resp = Response()
    session = get_session_id()

    log.info("session=%s POST /unlock", session)

    # don't do uneccesary unlocks
    if not _vault.is_locked():
        log.debug("session=%s agent is already unlocked, ignoring", session)
        return ""

    try:
        data = request.json
        if not _vault.exists():
            log.info("session=%s key vault does not exist, creating", session)
            _vault.create(data['passphrase'])
        _vault.unlock(data['passphrase'])
        if not _vault.is_locked():
            log.info("session=%s key vault successfuly unlocked", session)
            resp.status_code = 200
        else:
            log.warning(
                "session=%s key vault unlock failed, wrong password", session)
            resp.status_code = 403
    except:
        log.error("session=%s key vault unlock exception", session)
        resp.status_code = 500

    return resp

@app.route("/ping", methods=['GET'])
def ping():
    return json.dumps({"pong": _vault.is_locked()})

@app.route("/retrieve", methods=['GET'])
def get():
    resp = Response()
    session = get_session_id()

    log.info("session=%s GET /retrieve", session)

    try:
        data = request.json
        log.info("session=%s retrieving data for key '"+data['id']+"'", session)
        secret = _vault.retrieve(data['id'])
        resp.status_code = 200
        resp.data = str(secret)
    except:
        log.error("session=%s key vault retrieval exception", session)
        resp.status_code = 500
    return resp


@app.route("/store", methods=['PUT'])
def put():
    resp = Response()
    session = get_session_id()

    log.info("session=%s PUT /store", session)

    try:
        data = request.json
        log.info("session=%s storing data for key '"+data['id']+"'", session)
        _vault.store(data['id'], data['key'], cipher="aes-jumpi")
        resp.status_code = 200
    except:
        log.error("session=%s key vault storage exception", session)
        resp.status_code = 500
    return resp
