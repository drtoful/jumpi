#-*- coding: utf-8 -*-

from flask import Flask
from jumpi.config import JumpiConfig

def create_app(modules="MODULES"):
    config = JumpiConfig()

    root = ""
    if config.ROOT_ENDPOINT.strip("/") != "":
        root = "/" + config.ROOT_ENDPOINT.strip("/")

    app = Flask(__name__, static_url_path=root+"/static")
    app.config.from_object(config)

    for module, prefix in app.config[modules]:
        module, attribute = module.rsplit('.', 1)

        try:
            _import = __import__(module, globals(), locals(), [attribute], -1)
            prefix = "" if prefix.strip('/') == "" else "/" + prefix.strip('/')
            prefix = root+prefix
            app.register_blueprint(
                getattr(_import, attribute),
                url_prefix=prefix
            )
        except Exception as e:
            print repr(e)

    return app
