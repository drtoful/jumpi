#-*- coding: utf-8 -*-

import os

from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy import Column, Integer, Sequence, String, DateTime
from sqlalchemy import ForeignKey, create_engine
from sqlalchemy.orm import sessionmaker, backref, relationship

_Base = declarative_base()

class User(_Base):
    __tablename__ = 'users'

    id = Column(Integer, Sequence('user_id_seq'), primary_key=True)
    fullname = Column(String, nullable=False)
    ssh_key = Column(String, nullable=False)
    ssh_fingerprint = Column(String, nullable=False)
    time_added = Column(DateTime(timezone="UTC"), nullable=False)
    time_lastaccess = Column(DateTime(timezone="UTC"))

    permissions = relationship("Permission",
        order_by="Permission.id", backref="users")

class Target(_Base):
    __tablename__ = 'targets'

    id = Column(String, primary_key=True)
    port = Column(Integer, nullable=False)
    type = Column(String, nullable=False)

    permissions = relationship("Permission",
        order_by="Permission.id", backref="targets")

class Permission(_Base):
    __tablename__ = 'permissions'

    user_id = Column(Integer, ForeignKey("users.id"))
    target_id = Column(String, ForeignKey("targets.id"))

    id = Column(Integer, Sequence('target_id_seq'), primary_key=True)
    user = relationship("User", backref=backref('users', order_by=id))
    target = relationship("Target", backref=backref('targets', order_by=id))

_home = os.path.expanduser("~")
_db_engine = create_engine("sqlite:///%s" % os.path.join(_home, "jumpi.db"))
_Base.metadata.create_all(_db_engine, checkfirst=True)

Session = sessionmaker(bind=_db_engine)
