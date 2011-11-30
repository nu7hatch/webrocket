#!/bin/bash

fail() {
	echo -e "\nERROR: Installation failed!"
	exit 1
}

echo "Installing awesomness of the WebRocket!"

echo -e "\n--- building webrocket library"
(make && make test && make install) || fail

echo -e "\n--- building rocket-server tool"
(cd rocket-server && make && make install) || fail
cd ..

echo -e "\nSuccess!"