#-*- coding: utf-8 -*-

import os
import json

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
        backref="user_targets", lazy=True)
    tunnel_permissions = relationship("TunnelPermission",
        order_by="TunnelPermission.id", cascade="all,delete",
        backref="user_tunnels", lazy=True)
    recordings = relationship("Recording",
        order_by="Recording.time.desc()", cascade="all, delete",
        backref="user_recordings", lazy=True)
    files = relationship("File", order_by="File.filename", cascade="all,delete",
        backref="user_files", lazy=True)

    def as_json(self):
        return dict(
            id = self.id,
            fullname = self.fullname,
            ssh_fingerprint = self.ssh_fingerprint,
            time_added = str(self.time_added),
            time_lastaccess = str(self.time_lastaccess)
        )

class Target(_Base):
    __tablename__ = 'targets'

    id = Column(String, primary_key=True)
    port = Column(Integer, nullable=False)
    type = Column(String, nullable=False)

    permissions = relationship("TargetPermission",
        order_by="TargetPermission.id", cascade="all,delete", backref="targets",
        lazy=True)

    def as_json(self):
        return dict(
            id = self.id,
            port = self.port,
            type = self.type
        )

class Recording(_Base):
    __tablename__ = 'recordings'

    id = Column(Integer, Sequence('recording_id_seq'), primary_key=True)
    user_id = Column(Integer, ForeignKey("users.id"))
    session_id = Column(String, nullable=False)
    width = Column(Integer, nullable=False, default=80)
    height = Column(Integer, nullable=False, default=24)
    duration = Column(Integer, nullable=False, default=0)
    time = Column(DateTime(timezone="UTC"), nullable=False)

    user = relationship("User", backref=backref('user_recordings', order_by=id,
        lazy='subquery'))

class File(_Base):
    __tablename__ = 'files'

    filename = Column(String, nullable=False, primary_key=True)
    user_id = Column(Integer, ForeignKey("users.id"))
    basename = Column(String, nullable=False)
    created = Column(DateTime(timezone="UTC"), nullable=False)
    size = Column(Integer, nullable=False)

    user = relationship("User",
        backref=backref('user_files', order_by=filename, lazy='subquery'))

    def as_json(self):
        return dict(
            filename = self.filename,
            user_id = self.user_id,
            basename = self.basename,
            created = str(self.created),
            size = self.size
        )

class TargetPermission(_Base):
    __tablename__ = 'target_permissions'

    id = Column(Integer, Sequence('target_permission_seq'), primary_key=True)
    user_id = Column(Integer, ForeignKey("users.id"))
    target_id = Column(String, ForeignKey("targets.id"))

    user = relationship("User", backref=backref('user_targets', order_by=id,
        lazy='subquery'))
    target = relationship("Target", backref=backref('targets', order_by=id,
        lazy='subquery'))

    def as_json(self):
        return dict(
            id = self.id,
            user_id = self.user_id,
            target_id = self.target_id
        )

class Tunnel(_Base):
    __tablename__ = 'tunnels'

    id = Column(Integer, Sequence('tunnel_id_seq'), primary_key=True)
    destination = Column(String, nullable=False)
    port = Column(Integer, nullable=False)

    permissions = relationship("TunnelPermission",
        order_by="TunnelPermission.id", cascade="all,delete",
        backref="tunnels", lazy=True)

class TunnelPermission(_Base):
    __tablename__ = 'tunnel_permissions'

    id = Column(Integer, Sequence('tunnel_permission_seq'), primary_key=True)
    user_id = Column(Integer, ForeignKey("users.id"))
    tunnel_id = Column(Integer, ForeignKey("tunnels.id"))

    user = relationship("User", backref=backref('user_tunnels', order_by=id,
        lazy='subquery'))
    tunnel = relationship("Tunnel", backref=backref('tunnels', order_by=id,
        lazy='subquery'))

try:
    import pwd
    _home = pwd.getpwuid(os.getuid()).pw_dir
except:
    _home = os.path.expanduser("~")

_dburl = "sqlite:///%s" % os.path.join(_home, "jumpi.db")
_db_engine = create_engine(_dburl)
Session = sessionmaker(bind=_db_engine)

# alembic operations
def db_create():
    from alembic.config import Config
    from alembic import command
    from pkg_resources import Requirement, resource_filename

    _Base.metadata.create_all(_db_engine)

    filename = resource_filename(Requirement.parse("jumpi"),"alembic.ini")
    cfg = Config(filename)
    cfg.set_main_option("sqlalchemy.url", _dburl)
    command.stamp(cfg, "head")

def db_upgrade():
    from alembic.config import Config
    from alembic import command
    from pkg_resources import Requirement, resource_filename

    filename = resource_filename(Requirement.parse("jumpi"),"alembic.ini")
    cfg = Config(filename)
    cfg.set_main_option("sqlalchemy.url", _dburl)
    command.upgrade(cfg, "head")

def db_migrate():
    from alembic.config import Config
    from alembic import command
    from pkg_resources import Requirement, resource_filename

    filename = resource_filename(Requirement.parse("jumpi"),"alembic.ini")
    cfg = Config(filename)
    cfg.set_main_option("sqlalchemy.url", _dburl)
    command.revision(cfg, autogenerate=True)

