package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"os"
	"io/ioutil"
	"log"
)
const initdb_sql_filename string = "./init_database.sql";
const functions_sql_filename string = "./functions.sql";
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

	// Read() PLSQL (stored procedures) 
    _, err = os.Stat(functions_sql_filename)
    if os.IsNotExist(err) {
		std_err.Println("Please place the file named 'functions.sql' into the current direcotry, this file must contain function definitions")
        os.Exit(2);
    }
	var plsql_data []byte;
	data,err=ioutil.ReadFile(functions_sql_filename);
	if (err!=nil) {
		std_err.Println("cant read ",functions_sql_filename);
		log.Fatal(err);
		os.Exit(2);
	}
	plsql_sql:=string(plsql_data);

	conn_str:="user='"+os.Getenv("ETHBOT_USERNAME")+"' dbname='"+os.Getenv("ETHBOT_DATABASE")+"' password='"+os.Getenv("ETHBOT_PASSWORD")+"' host='"+os.Getenv("ETHBOT_HOST")+"'";

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
	_,err=db.Exec(plsql_sql);
	if (err!=nil) {
		log.Fatal(err);
	} else {
		std_out.Println("SQL functions (PLSQL) have been created");
	}
}

