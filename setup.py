#-*- coding: utf-8 -*-

from setuptools import setup

setup(
    name="jumpi",
    version="0.1",
    packages=['jumpi', 'jumpi.agent', 'jumpi.web', 'jumpi.sh'],
    package_dir={'jumpi': 'jumpi'},
    package_data={'jumpi': ['web/templates/*', 'web/static/*.css']},
    include_package_data=True,
    zip_safe=False,
    install_requires=[
        'flask >= 0.9',
        'python-daemon >= 1.6',
        'sqlalchemy >= 0.9',
        'requests >= 2.2.1',
        'paramiko >= 1.14',
        'pyvault>=0.1'
    ],
    dependency_links=[
        'https://github.com/drtoful/pyvault/tarball/0.1#egg=pyvault-0.1git'
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
            'jumpi-sh = jumpi.sh.main:main',
            'jumpi-agent = jumpi.agent.main:Main.run',
            'jumpi-web = jumpi.web.main:Main.run'
        ],
    },
)

