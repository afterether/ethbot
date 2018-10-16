/*
	Copyright 2018 The AfterEther Team
	This file is part of the EthBot, Ethereum Blockchain -> SQL converter.
		
	EthBot is free software: you can redistribute it and/or modify
	it under the terms of the GNU Lesser General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
	
	EthBot is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
	GNU Lesser General Public License for more details.
	
	You should have received a copy of the GNU Lesser General Public License
	along with EthBot. If not, see <http://www.gnu.org/licenses/>.
*/
package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"os"
	"io/ioutil"
	"log"
	"net"
)
const initdb_sql_filename string = "./init_database.sql";
func main() {
	var err error
	var db *sql.DB
	std_err:=log.New(os.Stderr,"",0)
	std_out:=log.New(os.Stdout,"",0)

	/// Read() table defintions and indexes
    _, err = os.Stat(initdb_sql_filename)
    if os.IsNotExist(err) {
		std_err.Println("Please place the file named 'init_database.sql' into the current direcotry, this file must contain the datbase initialization commands")
        os.Exit(2);
    }
	var data []byte;
	data,err=ioutil.ReadFile(initdb_sql_filename);
	if (err!=nil) {
		std_err.Println("cant read ",initdb_sql_filename);
		log.Fatal(err);
		os.Exit(2);
	}
	initdb_sql:=string(data);

	host,port,err:=net.SplitHostPort(os.Getenv("ETHBOT_HOST"))
	if (err!=nil) {
		host=os.Getenv("ETHBOT_HOST")
		port="5432"
	}
	conn_str:="user='"+os.Getenv("ETHBOT_USERNAME")+"' dbname='"+os.Getenv("ETHBOT_DATABASE")+"' password='"+os.Getenv("ETHBOT_PASSWORD")+"' host='"+host+"' port='"+port+"'";
	db,err=sql.Open("postgres",conn_str);
	if (err!=nil) {
		log.Fatal(err);
	} else {
	}
	_,err=db.Exec(initdb_sql);
	if (err!=nil) {
		log.Fatal(err);
	} else {
		std_out.Println("Database has been initialized");
	}
}

