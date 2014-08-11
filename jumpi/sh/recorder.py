#-*- coding: utf-8 -*-

#
# Based upon the python implementation of the asciinema
# recording utility, created by Marcin Kulik. Licensed
# under MIT License.
#
# check original code under:
#   https://github.com/asciinema/asciinema-cli/blob/524f1dde9709a4c15b90721c9e94a8716fcdfb09/asciinema/pty_recorder.py
#

import pyte
import json
import pty
import sys
import os
import signal
import array
import fcntl
import termios
import errno
import select
import time
import subprocess
import tty
import base64

def _force_unicode(txt):
    try:
        return unicode(txt)
    except UnicodeDecodeError:
        pass

    orig = txt
    if type(txt) != str:
        txt = str(txt)
    for args in [('utf-8',), ('latin1',), ('ascii', 'replace')]:
        try:
            return txt.decode(*args)
        except UnicodeDecodeError:
            pass

    return ""

class Recorder(object):
    class Recording(object):
        def __init__(self, lines, columns):
            self.screen = pyte.DiffScreen(columns, lines)
            self.stream = pyte.Stream()
            self.lines = lines
            self.columns = columns
            self.recordings = []
            self.duration = 0

            self.stream.attach(self.screen)

        def update(self, capture):
            secs, microsecs, data = capture

            self.screen.dirty.clear()
            self.stream.feed(_force_unicode(data))
            display = self.screen.display

            self.duration += secs
            self.duration += microsecs/1000000.0

            self.recordings.append({
                'delay': secs*1000000+microsecs,
                'changes': list(self.screen.dirty),
                'data': [display[x].rstrip() for x in self.screen.dirty],
                'raw': base64.b64encode(data)
            })

        def __str__(self):
            return json.dumps({
                'duration': self.duration,
                'size': {'lines': self.lines, 'columns': self.columns},
                'recording': self.recordings
            })

    def _set_pty_size(self):
        """setting size of virtual terminal"""
        # Get the terminal size of the real terminal, set it on the
        # pseudoterminal
        if os.isatty(pty.STDOUT_FILENO):
            self.buf = array.array('h', [0, 0, 0, 0])
            fcntl.ioctl(pty.STDOUT_FILENO, termios.TIOCGWINSZ, self.buf, True)
            fcntl.ioctl(self.master_fd, termios.TIOCGWINSZ, self.buf)
        else:
            self.buf = array.array('h', [24, 80, 0, 0])
            fcntl.ioctl(self.master_fd, termios.TIOCGWINSZ, self.buf)

    def _signal_winch(self, signal, frame):
        """signal handler for window size change"""
        self._set_pty_size()

    def _write_master(self, data):
        """writes to the child process from controlling terminal"""
        while data:
            n = os.write(self.master_fd, data)
            data = data[n:]

    def _write_stdout(self, data):
        """writes to stdout as if the child process had written the data"""
        os.write(pty.STDOUT_FILENO, data)

    def _handle_stdin_read(self, data):
        """handles new data on child process stdin."""
        self._write_master(data)

    def _handle_master_read(self, data):
        """handles new data on child process stdout."""
        self._write_stdout(data)

        now = time.time()
        delta = now - self.timing
        self.timing = now

        secs, microsecs = ("%f" % delta).split(".")
        self.output.append((int(secs), int(microsecs), data))

    def _copy(self):
        """main select loop"""
        while True:
            try:
                rfds, wfds, xfds = select.select(
                    [self.master_fd, pty.STDIN_FILENO], [], [])
            except select.error as e:
                # interrupted system call
                if e[0] == errno.EINTR:
                    continue

            if self.master_fd in rfds:
                data = os.read(self.master_fd, 1024)
                if len(data) == 0:
                    break

                self._handle_master_read(data)

            if pty.STDIN_FILENO in rfds:
                data = os.read(pty.STDIN_FILENO, 1024)
                self._handle_stdin_read(data)

    def record(self, func, *args, **kwargs):
        pid, self.master_fd = pty.fork()
        self.timing = time.time()
        self.output = []

        # child is executing original program
        if pid == pty.CHILD:
            def signal_sigint(signal, frame):
                sys.exit(0)
            signal.signal(signal.SIGINT, signal_sigint)

            func(*args, **kwargs)
            sys.exit(0)

        # parent is capturing screen of child

        # get all window size changes
        old_handler = signal.signal(signal.SIGWINCH, self._signal_winch)

        try:
            mode = tty.tcgetattr(pty.STDIN_FILENO)
            tty.setraw(pty.STDIN_FILENO)
            restore = True
        except tty.error:
            # this is the same as termios error
            restore = False

        self._set_pty_size()

        try:
            self._copy()
        except (IOError, OSError):
            if restore:
                tty.tcsetattr(pty.STDIN_FILENO, tty.TCSAFLUSH, mode)

        os.close(self.master_fd)
        signal.signal(signal.SIGWINCH, old_handler)

        def get_command_output(args):
            process = subprocess.Popen(args, stdout=subprocess.PIPE)
            return process.communicate()[0].strip()

        try:
            lines = int(get_command_output(["tput", "lines"]))
        except:
            lines = 24

        try:
            cols = int(get_command_output(["tput", "cols"]))
        except:
            cols = 80

        self.recording = Recorder.Recording(lines, cols)
        for capture in self.output:
            self.recording.update(capture)
