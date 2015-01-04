#-*- coding: utf-8 -*-

from setuptools import setup
from distutils.command.install import INSTALL_SCHEMES

for scheme in INSTALL_SCHEMES.values():
    scheme['data'] = scheme['purelib']

extras = {
    'with_otp_google': [
        'pyotp >= 1.3.0',
        'qrcode >= 5.1'
    ],
    'with_otp_yubico': [
        'yubico-client >= 1.9.1'
    ],
    'with_pyte': [
        'pyte >= 0.4.8',
    ]
}

setup(
    name="jumpi",
    version="0.1",
    packages=['jumpi', 'jumpi.agent', 'jumpi.web', 'jumpi.sh', 'alembic'],
    package_dir={'jumpi': 'jumpi'},
    package_data={
        'jumpi': ['web/templates/*', 'web/static/*'],
        'alembic': ['alembic/versions/*.py'],
    },
    data_files=[('',['alembic.ini'])],
    include_package_data=True,
    zip_safe=False,
    install_requires=[
        'flask >= 0.9',
        'sqlalchemy >= 0.9',
        'requests >= 2.2.1,<2.3',
        'paramiko >= 1.14',
        'pyvault >= 0.2.1',
        'alembic >= 0.6.7',
    ],
    extras_require=extras,
    dependency_links=[
        'https://github.com/drtoful/pyvault/tarball/0.2.1#egg=pyvault-0.2.1git'
    ],

    # package metadata
    author="Tobias Heinzen",
    description="simple administration for ssh jumphost",
    long_description="""

JumPi is a collection of utilities to create a simple SSH Jumphost.

You can define targets that should be accessible and assign them to
users to create a simple access control.

    """.strip(),
    license="BSD",
    classifiers=[
        'Development Status :: 3 - Alpha',
        'Intended Audience :: System Administrators',
        'License :: OSI Approved :: BSD License',
        'Topic :: Security',
    ],

    # cmd line scripts
    entry_points = {
        'console_scripts': [
            'jumpish = jumpi.sh.main:main',
            'jumpidb-create = jumpi.db:db_create',
            'jumpidb-upgrade = jumpi.db:db_upgrade'
        ],
    },
)

