#-*- coding: utf-8 -*-

import sys
import os
import datetime

from jumpi.sh.agent import Agent
from jumpi.sh.shell import JumpiShell
from jumpi.db import Session, User, Recording
from jumpi.sh import log
from jumpi.sh.recorder import Recorder

def main():
    # check argument length
    if len(sys.argv) != 2:
        print >>sys.stderr, "usage: %s <id>" % (sys.argv[0])
        return

    # check if users exists
    session = Session()
    user = session.query(User).filter_by(id=sys.argv[1]).first()
    if user is None:
        print >>sys.stderr, "user not found!"
        return

    # check if agent is up and running
    a = Agent()
    (resp, reason) = a.ping()
    if not resp:
        log.error("user='%s' tried to log in, but agent is locked" %(
            user.fullname))
        print >>sys.stderr, reason
        return

    user.time_lastaccess = datetime.datetime.now()
    session.merge(user)
    session.commit()

    intro = """Welcome to JumPi Interactive Shell!
You're logged in as: %s
""" % (user.fullname)

    shell = JumpiShell(user)
    recorder = Recorder()

    cmd = os.environ.get('SSH_ORIGINAL_COMMAND', None)
    if cmd is None:
        recorder.record(shell.cmdloop, intro=intro)
    else:
        recorder.record(shell.onecmd, cmd)

    # save the recording to the users recordings list
    recording = Recording(
        user_id = user.id,
        session_id = shell.session,
        duration = recorder.recording.duration,
        width = recorder.recording.columns,
        height = recorder.recording.lines,
        time = datetime.datetime.now()
    )
    if a.store_data(str(user.id)+"@"+shell.session, str(recorder.recording)):
        session.add(recording)
        session.commit()
