#!/bin/bash

SRC=$GOPATH/src/github.com/afterether/ethbot/vendor/github.com/ethereum/go-afterether-1.8.12
DST=$GOPATH/src/github.com/afterether/ethbot/vendor/github.com/ethereum/go-ethereum
ln -snrf $SRC $DST
ls -l $DST/../
