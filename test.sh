#!/bin/sh

set -e

if ! [ -f /tmp/GSR2.crt ]; then
  curl -so /tmp/GSR2.crt https://pki.goog/gsr2/GSR2.crt
fi

if ! [ -f /tmp/goog.pem ]; then
  openssl x509 -inform der -in /tmp/GSR2.crt -out /tmp/goog.pem
fi

if ! go run certswap.go --ca /tmp/goog.pem -- curl -sf https://google.com/humans.txt > /dev/null 2>&1 ; then
  echo "expected curl-ing google to work"
  exit 1
fi

if go run certswap.go --ca /tmp/goog.pem -- curl -sf https://bing.com > /dev/null 2>&1 ; then
  echo "expected curl-ing bing to fail"
  exit 1
fi

echo "success"
