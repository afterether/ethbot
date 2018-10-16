#!/bin/bash

cat tables.sql indexes.sql inserts.sql trigger-funcs.sql triggers.sql functions.sql > init_database.sql
