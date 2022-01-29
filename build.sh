#!/bin/sh
go clean .
go build .

DST="./release"
rm -rf $DST
mkdir  $DST
mkdir  $DST/tls
mkdir  $DST/records
cp     tls/README.md $DST/tls/README.md
cp     witty LICENSE $DST
echo   "[]" > $DST/user.db