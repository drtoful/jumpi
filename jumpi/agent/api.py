#-*- coding: utf:8 -*-

import json
import os

from flask import Flask, request, Response
from pyvault import PyVault
from pyvault.backends.file import PyVaultFileBackend

app = Flask(__name__)

_backend = PyVaultFileBackend(
    os.path.join(os.path.expanduser("~"), ".store")
)
_vault = PyVault(_backend)

@app.route("/unlock", methods=['POST'])
def unlock():
    resp = Response()

    # don't do uneccesary unlocks
    if not _vault.is_locked():
        return ""

    try:
        data = request.json
        if not _vault.exists():
            _vault.create(data['passphrase'])
        _vault.unlock(data['passphrase'])
        if not _vault.is_locked():
            resp.status_code = 200
        else:
            resp.status_code = 403
    except:
        resp.status_code = 500

    return resp

@app.route("/ping", methods=['GET'])
def ping():
    return json.dumps({"pong": _vault.is_locked()})

@app.route("/retrieve", methods=['GET'])
def get():
    resp = Response()

    try:
        data = request.json
        secret = _vault.retrieve(data['id'])
        resp.status_code = 200
        resp.data = str(secret)
    except:
        resp.status_code = 500
    return resp


@app.route("/store", methods=['PUT'])
def put():
    resp = Response()

    try:
        data = request.json
        _vault.store(data['id'], data['key'])
        resp.status_code = 200
    except:
        resp.status_code = 500
    return resp

