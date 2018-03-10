package main

import (
	"os"
	"log"
	"strconv"
    "github.com/syndtr/goleveldb/leveldb"
	"encoding/binary"
	"bytes"
    "github.com/syndtr/goleveldb/leveldb/opt"
)

func get_last_block(db *leveldb.DB) (uint64,error) {
    data, err := db.Get([]byte("last_block"),nil)
    if err != nil {
        return 0, err
    }

    var value uint64
    err = binary.Read(bytes.NewReader(data), binary.LittleEndian, &value)
    if err != nil {
        return 0, err
    }

    return value, nil
}
func put_last_block(db *leveldb.DB,block uint64) error {
    buf := new(bytes.Buffer)
    err := binary.Write(buf, binary.LittleEndian, block)
    if err != nil {
        return err
    }
    return db.Put([]byte("last_block"), buf.Bytes(),nil)
}
func main() {

	std_err:=log.New(os.Stderr,"",0)
	std_out:=log.New(os.Stdout,"",0)

	if (len(os.Args)<2) {
		std_err.Println("usage: ",os.Args[0]," [path_to_db] (optional_new_value_for_last_block_to_set)" );
		std_err.Println("\t`last_block` sets the variable of the last processed block to some value, use it to restart block processing from some block number" );
		std_err.Println("\tto run EthBot from genesis block set `last_block` to -1" );
		os.Exit(2);
	}

	var opts opt.Options
	opts.ErrorIfMissing=true
	db, err := leveldb.OpenFile(os.Args[1],&opts)
    if err != nil {
		std_err.Printf("can't open database (bad path to DB, corrupt data or busy by another `geth` process) error=%v",err)
        os.Exit(1)
    }

	last_block,err:=get_last_block(db)
	if (err != nil) {
		std_err.Println("error getting 'last_block' key from database, probably it is not set because you didn't run EthBot yet");
	} else {
		std_out.Println("last_block =",last_block)
	}
	var new_last_block_num uint64 = 0;
	if (len(os.Args)==3) {
		tmpval,err:=strconv.ParseInt(os.Args[2],10,64);
		if (err!=nil) {
			std_out.Println("error parsing `last_block` parameter, it must be a number");
			os.Exit(3);
		}
		if (tmpval==-1) {
			std_out.Println("clearing `last_block` variable, now EthBot will start processing blocks from block 0, the genesis block")
			db.Delete([]byte("last_block"),nil)
		} else {
			new_last_block_num=uint64(tmpval);
			std_out.Println("setting new value to `last_block`: last_block =",new_last_block_num);
			put_last_block(db,new_last_block_num);
		}
	}

	db.Close();

	os.Exit(0)
}
