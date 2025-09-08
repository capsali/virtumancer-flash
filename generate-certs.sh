#!/bin/bash
# This script generates a self-signed SSL certificate for local development.

# Exit immediately if a command exits with a non-zero status.
set -e

# The name for the key and certificate files.
FILENAME="localhost"

# Check if openssl is installed.
if ! [ -x "$(command -v openssl)" ]; then
  echo 'Error: openssl is not installed.' >&2
  exit 1
fi

# Generate the private key and certificate.
# -x509: outputs a self-signed certificate instead of a certificate request.
# -newkey rsa:4096: creates a new 4096-bit RSA key.
# -keyout: the file to write the private key to.
# -out: the file to write the certificate to.
# -sha256: use 256-bit SHA to sign the certificate.
# -days 3650: the certificate will be valid for 10 years.
# -nodes: don't encrypt the private key.
# -subj: sets the subject name for the certificate.
openssl req -x509 -newkey rsa:4096 \
  -keyout "${FILENAME}.key" \
  -out "${FILENAME}.crt" \
  -sha256 \
  -days 3650 \
  -nodes \
  -subj "/C=XX/ST=State/L=City/O=Organization/OU=Unit/CN=localhost"

echo "✅ Certificate and private key generated: ${FILENAME}.crt, ${FILENAME}.key"
echo "ℹ️  You may need to trust the '${FILENAME}.crt' file in your browser or operating system."
