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

    target_permissions = relationship("TargetPermission",
        order_by="TargetPermission.id", cascade="all,delete",
        backref="user_targets")
    tunnel_permissions = relationship("TunnelPermission",
        order_by="TunnelPermission.id", cascade="all,delete",
        backref="user_tunnels")

class Target(_Base):
    __tablename__ = 'targets'

    id = Column(String, primary_key=True)
    port = Column(Integer, nullable=False)
    type = Column(String, nullable=False)

    permissions = relationship("TargetPermission",
        order_by="TargetPermission.id", cascade="all,delete", backref="targets")

class TargetPermission(_Base):
    __tablename__ = 'target_permissions'

    id = Column(Integer, Sequence('target_permission_seq'), primary_key=True)
    user_id = Column(Integer, ForeignKey("users.id"))
    target_id = Column(String, ForeignKey("targets.id"))

    user = relationship("User", backref=backref('user_targets', order_by=id))
    target = relationship("Target", backref=backref('targets', order_by=id))

class Tunnel(_Base):
    __tablename__ = 'tunnels'

    id = Column(Integer, Sequence('tunnel_id_seq'), primary_key=True)
    destination = Column(String, nullable=False)
    port = Column(Integer, nullable=False)

    permissions = relationship("TunnelPermission",
        order_by="TunnelPermission.id", cascade="all,delete",
        backref="tunnels")

class TunnelPermission(_Base):
    __tablename__ = 'tunnel_permissions'

    id = Column(Integer, Sequence('tunnel_permission_seq'), primary_key=True)
    user_id = Column(Integer, ForeignKey("users.id"))
    tunnel_id = Column(Integer, ForeignKey("tunnels.id"))

    user = relationship("User", backref=backref('user_tunnels', order_by=id))
    tunnel = relationship("Tunnel", backref=backref('tunnels', order_by=id))

_home = os.path.expanduser("~")
_db_engine = create_engine("sqlite:///%s" % os.path.join(_home, "jumpi.db"))
_Base.metadata.create_all(_db_engine, checkfirst=True)

Session = sessionmaker(bind=_db_engine)
