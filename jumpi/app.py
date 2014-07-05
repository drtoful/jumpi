#-*- coding: utf-8 -*-

import optparse
import sys
import daemon
import os
import signal
import subprocess

class DaemonApp(object):
    def __init__(self):
        self.name = sys.argv[0]

    def help(self):
        print >>sys.stderr, "usage: %s <start|stop|status>" % (self.name)

    def _get_pid(self):
        with open(self.pidfile, "r") as fp:
            pid = int("".join(fp.readlines()))
        return pid

    def status(self):
        if not os.path.isfile(self.pidfile):
            return False

        pid = self._get_pid()
        env = os.environ
        process = subprocess.Popen(
            'ps -p %d' % (pid), env=env, shell=True,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE
        )
        process.wait()

        if process.returncode == 0:
            return True

        return False

    def stop(self):
        if not self.status():
            return True

        pid = self._get_pid()
        os.kill(pid, signal.SIGKILL)
        os.remove(self.pidfile)

        return not self.status()

    def _start(self):
        if self.status():
            return True

        context = daemon.DaemonContext()
        context.prevent_core = True

        with context:
            pid = os.getpid()

            with open(self.pidfile, "w") as fp:
                print >>fp, str(pid)

            self.start()

        return self.status()


    def run(self):
        parser = optparse.OptionParser()
        (_, args) = parser.parse_args()

        if len(args) == 0:
            self.help()
            return

        if args[0] == "start":
            print "starting %s: " % (self.name),
            if self._start():
                print "OK"
            else:
                print "ERROR"
            return

        if args[0] == "stop":
            print "stopping %s: " % (self.name),
            if self.stop():
                print "OK"
            else:
                print "ERROR"
            return

        if args[0] == "status":
            print "status %s: " % (self.name),
            if self.status():
                print "up"
            else:
                print "down"
            return

