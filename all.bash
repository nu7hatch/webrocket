#!/bin/bash

fail() {
	echo -e "\nERROR: Installation failed!"
	exit 1
}

echo "Installing awesomness of the WebRocket!"

echo -e "\n--- building webrocket library"
(cd webrocket && make && make test && make install) || fail

echo -e "\n--- building webrocket-server tool"
(cd webrocket-server && make && make install) || fail
cd ..

echo -e "\nSuccess!"