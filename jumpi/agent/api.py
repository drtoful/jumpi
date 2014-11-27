#-*- coding: utf:8 -*-

import json
import datetime
import os
import ConfigParser
import pyotp

from flask import Blueprint, request
from pyvault import PyVault
from pyvault.backends.ptree import PyVaultPairtreeBackend
from pyvault.ciphers.aes import PyVaultCipherAES
from pyvault.ciphers import cipher_manager
from jumpi.agent import log, get_session_id
from jumpi.config import HOME_DIR, get_config, JumpiConfig
from jumpi.db import Session, User, Recording, File, Target
from jumpi.agent.utils import json_validate, json_required
from jumpi.agent.utils import compose_json_response

app = Blueprint("base", __name__)

class _JumpiAES(PyVaultCipherAES):
    def __init__(self):
        PyVaultCipherAES.__init__(self)

        config = get_config()
        self.KEYDERIV_ITERATIONS = config.getint(
            "cipher", "iterations", JumpiConfig.CIPHER_ITERATIONS)

    @property
    def id(self):
        return "aes-jumpi"

_backend = PyVaultPairtreeBackend(
    os.path.join(HOME_DIR, ".store")
)
_vault = PyVault(_backend)
cipher_manager.register("aes-jumpi", _JumpiAES())

@app.route("/vault/unlock", methods=['POST'])
@json_required()
@json_validate(required=["passphrase"], passphrase="string")
def unlock():
    session = get_session_id()

    log.info("session=%s POST /vault/unlock", session)

    # don't do uneccesary unlocks
    if not _vault.is_locked():
        log.debug("session=%s agent is already unlocked, ignoring", session)
        return compose_json_response(202, message="Agent already unlocked")

    created = False
    if not _vault.exists():
        log.info("session=%s key vault does not exist, creating", session)

        config = get_config()
        iterations = config.getint("vault", "iterations",
            JumpiConfig.VAULT_ITERATIONS)
        complexity = config.getint("vault", "complexity",
            JumpiConfig.VAULT_COMPLEXITY)

        try:
            _vault.create(data['passphrase'], complexity, iterations)
            created = True
        except:
            log.error("session=%s key vault creation failed", session)
            return compose_json_response(500, error="vault_creation",
                error_long="Vault could not be created")

    data = request.json
    try:
        _vault.unlock(data['passphrase'])
    except:
        log.error("session=%s key vault unlock failed", session)
        return compose_json_response(500, error="vault_unlock",
            error_long="Failure during vault unlock")

    if not _vault.is_locked():
        log.info("session=%s key vault successfuly unlocked", session)
        if created:
            return compose_json_response(201)
        return compose_json_response(200)
    else:
        log.warning(
            "session=%s key vault unlock failed, wrong password", session)
        return compose_json_response(403, error="vault_locked",
            error_long="Vault is locked; wrong passphrase provided")

    return compose_json_response(500, error="unreachable",
        error_long="Server reached unreachable method end")

@app.route("/vault/status", methods=['GET'])
def ping():
    return compose_json_response(200, locked=_vault.is_locked())

@app.route("/vault/retrieve", methods=['GET'])
@json_required()
@json_validate(required=["key"], key="string")
def get():
    session = get_session_id()

    if _vault.is_locked():
        return compose_json_response(423, error="vault_locked",
            error_long="Vault is locked; unlock first")

    data = request.json
    log.info("session=%s retrieving data for key '"+data['key']+"'", session)

    try:
        secret = _vault.retrieve(data['key'])
        return compose_json_response(200, value=str(secret))
    except ValueError:
        return compose_json_response(404, error="not_in_vault",
            error_long="Key not found in vault")
    except:
        pass

    log.error("session=%s key vault retrieval exception", session)
    return compose_json_response(500, error="vault_retrieve",
        error_long="Unable to retrieve data under key=%s in store" % data['key'])

@app.route("/vault/store", methods=['PUT'])
@json_required()
@json_validate(required=["key","value"], key="string", value="string")
def put():
    session = get_session_id()

    if _vault.is_locked():
        return compose_json_response(423, error="vault_locked",
            error_long="Vault is locked; unlock first")

    data = request.json
    log.info("session=%s storing data for key '"+data['key']+"'", session)
    try:
        _vault.store(data['key'], data['value'], cipher="aes-jumpi")
        return compose_json_response(200)
    except:
        pass

    log.error("session=%s key vault store exception", session)
    return compose_json_response(500, error="vault_store",
        error_long="Unable to store data under key=%s in store" % data['key'])

@app.route("/target", methods=['GET'])
@json_required()
@json_validate(required=["id"], id="string")
def target():
    try:
        data = request.json

        session = Session()
        target = session.query(Target).filter_by(id=data['id']).first()
        if target is None:
            return compose_json_response(404, error="target_not_found",
                error_long="Target under id=%s not found" % data['id'])
        else:
            return compose_json_response(200, **target.as_json())
    except:
        pass

    session = get_session_id()
    log.error("session=%s error when trying to get target data id=%s",
        session, data['id'])
    return compose_json_response(500, error="target_load",
        error_long="Unable to get target information")

@app.route("/user/info", methods=['GET'])
@json_required()
@json_validate(required=["user"], user="integer")
def user_info():
    try:
        data = request.json

        session = Session()
        user = session.query(User).filter_by(id=data['user']).first()
        if user is None:
            return compose_json_response(404, error="user_not_found",
                error_long="User with id=%d not found" % data['user'])
        else:
            return compose_json_response(200, **user.as_json())
    except:
        pass

    return compose_json_response(500, error="user_load",
        error_long="Unable to load user information")

@app.route("/user/info", methods=['PATCH'])
@json_required()
@json_validate(required=["user"], user="integer", time_lastaccess="date",
    twofactor="boolean")
def user_info_set():
    session = Session()
    session_id = get_session_id()
    data = request.json

    user = session.query(User).filter_by(id=data['user']).first()
    if user is None:
        return compose_json_response(404, error="user_not_found",
            error_long="User with id=%d not found" % data['user'])

    if not data.get('time_lastaccess', None) is None:
        user.time_lastaccess = datetime.datetime.strptime(
            data['time_lastaccess'], "%Y-%m-%d %H:%M:%S")

    if not data.get('twofactor', None) is None:
        user.twofactor = data['twofactor']

    try:
        session.merge(user)
        session.commit()

        return compose_json_response(200)
    except:
        pass

    log.error("session=%s unable to patch user=%d", data['user'])
    return compose_json_response(500, error="user_patch",
        error_long="Unable to patch user with provided parameters")

@app.route("/user/permissions", methods=['GET'])
@json_required()
@json_validate(required=["user"], user="integer")
def user_targets():
    session = Session()
    data = request.json

    user = session.query(User).filter_by(id=data['user']).first()
    if user is None:
        return compose_json_response(404, error="user_not_found",
            error_long="User with id=%d not found" % data['user'])

    return compose_json_response(200, permissions=[
        x.as_json() for x in user.target_permissions])

@app.route("/user/files", methods=['GET'])
@json_required()
@json_validate(required=['user'], user="integer")
def user_files():
    session = Session()
    data = request.json

    user = session.query(User).filter_by(id=data['user']).first()
    if user is None:
        return compose_json_response(404, error="user_not_found",
            error_long="User with id=%d not found" % data['user'])

    return compose_json_response(200, files=[
        x.as_json() for x in user.files])

@app.route("/file", methods=['DELETE'])
@json_required()
@json_validate(required=['filename'], filename="string")
def file_delete():
    session_id = get_session_id()
    data = request.json
    session = Session()

    file = session.query(File).filter_by(filename=data['filename']).first()
    if file is None:
        return compose_json_response(404, error="file_not_found",
            error_logn="File not found")

    session.delete(file)
    session.commit()

    return compose_json_response(200)

@app.route("/file", methods=['PUT'])
@json_required()
@json_validate(required=["filename", "basename", "user_id", "created", "size"],
    filename="string", basename="string", user_id="integer", created="date",
    size="integer")
def file_put():
    session_id = get_session_id()
    data = request.json

    file = File(
        user_id = data['user_id'],
        basename = data['basename'],
        filename = data['filename'],
        size = data['size'],
        created = datetime.datetime.now()
    )

    try:
        session = Session()
        session.merge(file)
        session.commit()
        log.info("session=%s adding file for user=%d id=%s",
            session_id, file.user_id, file.basename)

        return compose_json_response(200)
    except:
        log.error("session=%s error adding recording for user=%d",
            session_id, id)

    return compose_json_response(500, error="file_store",
        error_long="Unable to store file information")

@app.route("/recording", methods=['PUT'])
@json_required()
@json_validate(required=["user_id", "session_id", "duration", "width", "height",
    "time"], user_id="integer", session_id="string", duration="integer",
    width="integer", height="integer", time="date")
def user_recording():
    session_id = get_session_id()
    data = request.json

    recording = Recording(
        user_id = data['user_id'],
        session_id = data['session_id'],
        duration = data['duration'],
        width = data['width'],
        height = data['height'],
        time = datetime.datetime.strptime(data['time'], "%Y-%m-%d %H:%M:%S")
    )

    try:
        session = Session()
        session.add(recording)
        session.commit()

        log.info("session=%s adding recording for user=%d id=%s",
            session_id, data['user_id'], recording.session_id)
        return compose_json_response(200)
    except:
        log.error("session=%s error adding recording for user=%d",
            session_id, id)

    return compose_json_response(500, error="recording_store",
        error_long="Unable to store recording information")

