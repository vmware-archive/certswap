# certswap

> do the ol' switcheroo on your system certificate store

## about

Some commands operate in two modes: they either use the system certificate
store or they allow you to disable TLS verification entirely. This is often an
unhelpful dichotomy. I often have a development PKI constructed which is
perfectly valid but I don't want to mess with the system store.

This tool lets you replace the system certificates for a single command
invocation while leaving them pristine for the rest of your system.

## install

```
go get github.com/xoebus/certswap
```

## example

```
curl -O https://pki.goog/gsr2/GSR2.crt
openssl x509 -inform der -in GSR2.crt -out goog.pem

# works
certswap --ca goog.pem -- curl https://www.google.com/humans.txt

# fails
certswap --ca goog.pem -- curl https://www.bing.com
```

## limitations

Only works on Debian and Ubuntu at the moment (or any distro which keeps it
certificates in `/etc/ssl/certs`).
