#!/bin/bash

SRC=$GOPATH/src/github.com/afterether/ethbot/vendor/github.com/ethereum/go-ethereum-1.7.2
DST=$GOPATH/src/github.com/afterether/ethbot/vendor/github.com/ethereum/go-ethereum
ln -snrf $SRC $DST
