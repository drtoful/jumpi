#-*- coding: utf-8 -*-

import os

from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy import Column, Integer, Sequence, String, DateTime
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

_Base = declarative_base()

class User(_Base):
    __tablename__ = 'users'

    id = Column(Integer, Sequence('user_id_seq'), primary_key=True)
    fullname = Column(String, nullable=False)
    ssh_key = Column(String, nullable=False)
    ssh_fingerprint = Column(String, nullable=False)
    time_added = Column(DateTime(timezone="UTC"), nullable=False)
    time_lastaccess = Column(DateTime(timezone="UTC"))

_home = os.path.expanduser("~")
_db_engine = create_engine("sqlite:///%s" % os.path.join(_home, "jumpi.db"))
_Base.metadata.create_all(_db_engine, checkfirst=True)

Session = sessionmaker(bind=_db_engine)
