#!/bin/bash

BINARY_NAME="proman"
INSTALL_DIR="/usr/bin"

if [ "$(id -u)" -ne 0 ]; then
  echo "This script must be run with sudo or as root to install to ${INSTALL_DIR}." >&2
  exit 1
fi

if [ ! -f "${BINARY_NAME}" ]; then
    echo "Error: The '${BINARY_NAME}' binary was not found in the current directory." >&2
    exit 1
fi

echo "Installing ${BINARY_NAME} to ${INSTALL_DIR}..."

install -m 0755 "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"

if [ $? -eq 0 ]; then
  echo "'${BINARY_NAME}' installed successfully"
else
  echo "Installation failed." >&2
  exit 1
fi

exit 0
