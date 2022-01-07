Past lesson shows that a test cert hurts security (because
people just use it). Follow the steps below to create a 
self-sigend ECC cert by yourself.

```
# generate a private key for a curve
openssl ecparam -name prime256v1 -genkey -noout -out private-key.pem

# Create a self-signed certificate
openssl req -new -x509 -key private-key.pem -out cert.pem -days 360
```
