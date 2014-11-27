#-*- coding: utf-8 -*-

import sys
import signal
import os
import datetime

from jumpi.sh.agent import Vault, User, format_datetime
from jumpi.sh.shell import JumpiShell
from jumpi.sh import log
from jumpi.sh.recorder import Recorder

handling = False

def main():
    # check argument length
    if len(sys.argv) != 2:
        print >>sys.stderr, "usage: %s <id>" % (sys.argv[0])
        return

    # check if agent is up and running
    vault = Vault()
    if vault.is_locked():
        log.error("user='%s' tried to log in, but agent is locked" %(
            sys.argv[1]))
        print >>sys.stderr, "Agent is locked!"
        return

    # check if users exists
    user = User(sys.argv[1])
    if not user.is_valid():
        print >>sys.stderr, "User not found!"
        return

    user.update('time_lastaccess', format_datetime(datetime.datetime.now()))

    intro = """Welcome to JumPi Interactive Shell!
You're logged in as: %s
""" % (user.fullname)

    shell = JumpiShell(user)
    recorder = Recorder()
    cmd = os.environ.get('SSH_ORIGINAL_COMMAND', None)

    def sig_terminating(*args, **kwargs):
        global handling

        if handling:
            return

        # save the recording to the users recordings list
        handling = True
        recording = dict(
            user_id = user.id,
            session_id = shell.session,
            duration = int(recorder.recording.duration),
            width = int(recorder.recording.columns),
            height = int(recorder.recording.lines),
            time = format_datetime(start)
        )
        id = str(user.id)+"@"+shell.session
        vault.store(id, str(recorder.recording))
        user.add_recording(**recording)

        sys.exit(0)

    signal.signal(signal.SIGTERM, sig_terminating)
    signal.signal(signal.SIGHUP, sig_terminating)

    start = datetime.datetime.now()
    if cmd is None:
        recorder.record(shell.cmdloop, intro=intro)
    else:
        if cmd.startswith("scp"):
            shell.onecmd(cmd)
        else:
            recorder.record(shell.onecmd, cmd)

    if cmd is None or not cmd.startswith("scp"):
        sig_terminating()
