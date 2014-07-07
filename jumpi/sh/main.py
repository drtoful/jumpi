#-*- coding: utf-8 -*-

import sys

from jumpi.sh.agent import Agent

def main():
    a = Agent()
    (resp, reason) = a.ping()
    if not resp:
        print >>sys.stderr, reason
        return
