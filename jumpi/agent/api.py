#-*- coding: utf:8 -*-

import json
import datetime
import os
import ConfigParser

from flask import Blueprint, request, Response
from pyvault import PyVault
from pyvault.backends.ptree import PyVaultPairtreeBackend
from pyvault.ciphers.aes import PyVaultCipherAES
from pyvault.ciphers import cipher_manager
from jumpi.agent import log, get_session_id, HOME_DIR
from jumpi.db import Session, User, Recording, File, Target

app = Blueprint("base", __name__)

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

    file = os.path.join(HOME_DIR, "jumpi-agent.cfg")
    parser = None
    if os.path.isfile(file):
        parser = ConfigParser.SafeConfigParser()
        parser.read(file)

    try:
        data = request.json
        if not _vault.exists():
            log.info("session=%s key vault does not exist, creating", session)

            iterations = 1000
            complexity = 10
            if parser:
                if parser.has_option("vault", "iterations"):
                    iterations = parser.getint("vault", "iterations")
                if parser.has_option("vault", "complexity"):
                    complexity = parser.getint("vault", "complexity")

            _vault.create(data['passphrase'], complexity, iterations)
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

@app.route("/target", methods=['GET'])
def target():
    resp = Response()
    session = Session()
    try:
        data = request.json
        target = session.query(Target).filter_by(id=data['id']).first()
        if target is None:
            resp.status_code = 404
        else:
            resp.status_code = 200
            resp.data = target.as_json()
    except:
        resp.status_code = 500

    return resp

@app.route("/user/<int:id>/info", methods=['GET'])
def user_info(id):
    resp = Response()
    session = Session()
    user = session.query(User).filter_by(id=id).first()
    if user is None:
        resp.status_code = 404
    else:
        resp.status_code = 200
        resp.data = user.as_json()

    return resp

@app.route("/user/<int:id>/info", methods=['POST'])
def user_info_set(id):
    resp = Response()
    session = Session()
    session_id = get_session_id()
    user = session.query(User).filter_by(id=id).first()
    if user is None:
        resp.status_code = 500
    else:
        try:
            data = request.json
            for key in data.keys():
                if not hasattr(user, key):
                    continue

                value = data[key]
                if key == "time_added" or key == "time_lastaccess":
                    value = datetime.datetime.strptime(value.split(".")[0],
                        "%Y-%m-%d %H:%M:%S")
                setattr(user, key, value)
                log.info("session=%s updating info for user=%d key=%s value=%s",
                    session_id, id, key, value)

            session.merge(user)
            session.commit()
            resp.status_code = 200
        except:
            log.error("session=%s error updating info for user=%d",
                session_id, id)
            resp.status_code = 500

    return resp

@app.route("/user/<int:id>/targets", methods=['GET'])
def user_targets(id):
    resp = Response()
    session = Session()
    user = session.query(User).filter_by(id=id).first()
    if user is None:
        resp.status_code = 404
    else:
        resp.status_code = 200
        data = ",".join([x.as_json() for x in user.target_permissions])
        resp.data = "["+data+"]"

    return resp

@app.route("/user/<int:id>/files", methods=['GET'])
def user_files(id):
    resp = Response()
    session = Session()
    user = session.query(User).filter_by(id=id).first()
    if user is None:
        resp.status_code = 404
    else:
        resp.status_code = 200
        data = ",".join([x.as_json() for x in user.files])
        resp.data = "["+data+"]"

    return resp

@app.route("/user/<int:id>/files", methods=['PUT'])
def user_files_put(id):
    resp = Response()
    session_id = get_session_id()

    try:
        data = request.json

        file = File(
            user_id = data['user_id'],
            basename = data['basename'],
            filename = data['filename'],
            size = data['size'],
            created = datetime.datetime.now()
        )

        session = Session()
        session.add(recording)
        session.commit()
        log.info("session=%s adding file for user=%d id=%s",
            session_id, id, file.basename)

        resp.status_code = 200
    except:
        log.error("session=%s error adding recording for user=%d",
            session_id, id)
        resp.status_code = 500

    return resp

@app.route("/user/<int:id>/recording", methods=['PUT'])
def user_recording(id):
    resp = Response()
    session_id = get_session_id()

    try:
        data = request.json

        recording = Recording(
            user_id = data['user_id'],
            session_id = data['session_id'],
            duration = data['duration'],
            width = data['width'],
            height = data['height'],
            time = datetime.datetime.strptime(data['time'].split('.')[0],
                "%Y-%m-%d %H:%M:%S")
        )
        session = Session()
        session.add(recording)
        session.commit()

        log.info("session=%s adding recording for user=%d id=%s",
            session_id, id, recording.session_id)

        resp.status_code = 200
    except:
        log.error("session=%s error adding recording for user=%d",
            session_id, id)
        resp.status_code = 500

    return resp
