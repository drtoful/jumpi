#-*- coding: utf-8 -*-

from setuptools import setup
from distutils.command.install import INSTALL_SCHEMES

for scheme in INSTALL_SCHEMES.values():
    scheme['data'] = scheme['purelib']

setup(
    name="jumpi",
    version="0.1",
    packages=['jumpi'],
    package_dir={'jumpi': 'jumpi'},
    package_data={
        'jumpi': ['jumpi/templates/*', 'jumpi/static/*'],
    },
    include_package_data=True,
    zip_safe=False,
    install_requires=[
        'flask >= 0.9',
        'requests >= 2.2.1',
    ],

    # package metadata
    author="Tobias Heinzen",
    description="simple administration ui for ssh jumphost",
    classifiers=[
        'Development Status :: 3 - Alpha',
        'Intended Audience :: System Administrators',
        'License :: OSI Approved :: BSD License',
        'Topic :: Security',
    ],
)

