#-*- coding: utf-8 -*-

import os

from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy import Column, Integer, Sequence, String, create_engine
from sqlalchemy.orm import sessionmaker

_Base = declarative_base()

class User(_Base):
    __tablename__ = 'users'

    id = Column(Integer, Sequence('user_id_seq'), primary_key=True)
    fullname = Column(String)
    sshkey = Column(String)

_home = os.path.expanduser("~")
_db_engine = create_engine("sqlite:///%s" % os.path.join(_home, "jumpi.db"))
_Base.metadata.create_all(_db_engine, checkfirst=True)

Session = sessionmaker(bind=_db_engine)
