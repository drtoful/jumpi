#-*- coding: utf:8 -*-

import json
import os

from flask import Flask
from pyvault import PyVault
from pyvault.backends.file import PyVaultFileBackend

app = Flask(__name__)

_backend = PyVaultFileBackend(
    os.path.join(os.path.expanduser("~"), ".store")
)
_vault = PyVault(_backend)

@app.route("/unlock", methods=['POST'])
def unlock():
    pass

@app.route("/ping", methods=['GET'])
def ping():
    return json.dumps({"pong": _vault.is_locked()})

@app.route("/retrieve", methods=['GET'])
def get():
    pass

@app.route("/store", methods=['PUT'])
def put():
    pass
