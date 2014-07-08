#-*- coding: utf-8 -*-

from flask import Flask

def create_app():
    app = Flask(__name__)
    app.config.from_object('jumpi.web.config.JumpiConfig')

    for module, prefix in app.config['MODULES']:
        module, attribute = module.rsplit('.', 1)

        try:
            _import = __import__(module, globals(), locals(), [attribute], -1)
            prefix = "" if prefix.strip('/') == "" else "/" + prefix.strip('/')
            prefix = prefix
            app.register_blueprint(
                getattr(_import, attribute),
                url_prefix=prefix
            )
        except Exception as e:
            print repr(e)

    return app
