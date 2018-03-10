
# EthBot

## About
#### EthBot is the backend for the AfterEther blockchain Explorer.

It is a limited `geth` node with added functionality to export blockchain data. It will attach to the network and export incoming blocks & transactions into SQL database for later use by other applications.


## Features

  * Converts blockchan (NoSQL) database to Postgres SQL
  * Extracts blockchain data down to value transfers (also called *internal transations*, the transactions made between contracts inside the VM)
  * Written in **Go** language and is embedded into `geth`
  * SQL (as opposite to NoSQL) makes it easy to write custom queries and generate specialized reports, query examples are provided at the end of this document
  * Includes verification processes to make sure that your SQL data matches blockchain data at 100%
  * Re-export range of blocks, to rewrite existing blocks in SQL database
  * Support and issue resolution are provided by the developers of the AfterEther currency team as a part of the AfterEther project (a project to scale Ethereum by blockchain clustering)

## Status

Under development, not ready for production. Expecting beta release to be ready soon.

## Installation

The installation available by 3 different ways.

  1. Binary download
  2. From patched sources in our repository
  3. By downloading Ethereum sources and applying the patches yourself


## Installation by binary download

ToDo

## Installation by sources from AfterEther

Download the sources from our repository

	git clone https://github.com/afterether/ethbot

Set your GOPATH to ethbot sources

Build EthBot:
	
	go build github/afterether/ethbot

During compile time, you may be asked for Go dependencies,
install them with the following command

	go get -v -u github.com/package_name_here

After compilation finishes the `ethbot` executable will be in current directory

	ls -l ./ethbot

## Installation by patching Ethereum `geth`

ToDo

## Configuring EthBot

#### Create a Postgres database 

Install and configure postgres as usual. Once Postgres is installed , create a new postgres user and a new database , owned by this user. Here is an example of how to do it :

	bash$ su - postgres
	bash$ createuser ethbot
	bash$ createdb ethbot
	bash$ psql
	# now you have entered Postgres SQL console
	postgres# alter user ethbot with encrypted password '123456';
	postgres# grant all privileges on database ethbot to ethbot ;

Edit the script `initdb.sh` and modify the environment variables :

  * **ETHBOT_HOSTNAME** , the host where your Postgress database is running, you may optionally specify the port number in the format [`host`:`port`]
  * **ETHBOT_USERNAME** , the username for your Postgres database

  * **ETHBOT_PASSWORD** , the password for the user owening the database
  * **ETHBOT_DATABASE** , the database name for your Postgres database

Run the script to create tables and PLSQL functions:

	./initdb.sh
	
If you want to modify the database initialization script, all you have to do is to alter the `init_database.sql` file or the `functions.sql` file, which are located within the same directory as the `init_database` executable file. You can also run `init_database.sql` directly in Postgres.

#### Configure EthBot to use the newly created database

Set the the environment variables to connect to the database exactly as during creation of the database:

Example:

	bash$ export ETHBOT_HOSTNAME=localhost
	bash$ export ETHBOT_USERNAME=ethbot
	bash$ export ETHBOT_PASSWORD=123456
	bash$ export ETHBOT_DATABASE=ethbot

You may want to hide these variables in a bash_profile script and set apropriate permissions so it is not accessible to everybody


## Running EthBot (Exporting data to SQL)

To run EthBot and start exporting the blockchain data immediately type this in the console:


`./ethbot`

You will achieve best performance if the node is already synchronized with the network, otherwise blocks will be inserted into SQL database as they are being downloaded from the network.

The following JavaScript functions are provided to operate EthBot from `geth` console:

    ethbot.blockchainExportStart(starting_block,ending_block,verify)
        starting_block: block to start the export process from
            -1 : to start exporting process from the last block you have exported in previous sessions 
             0 : start from the GENESIS block
             1..MAX_INT: the integer value corresponding to the block number you want the export process to start from
        ending_block: the block to finish export at
            -1 : enter listening mode and export incoming blocks, after existing blocks have been exported
             0..MAX_INT: ending block to stop export process (inclusive)
        verify: verify balances in SQL database after exporting each block (mode=0 verification is used)
		Returns: boolean value
            true: export process has started
            false: export process failed to start

    ethbot.blockchainExportStop()
		Returns: boolean value
            true: request to stop exporting process has been accepted
            false: failure to stop exporting process
		

	ethbot.blockchainExportStatus()
		Returns: Object
			current_block: the block that is being currently exported
			direction: the direction of the export (1 up, block numbers are increasing, -1 down, block numbers are decreasing)
            ending_block: the last block to export (inclusive)
            listening_mode: true if after exporting all the blocks, EthBot has to enter into listening mode and export incoming blocks
            starting_block: the block to start the export from
            verify: execute verification process after exporting each block (mode=0 is used)

		Example:
		> ethbot.blockchainExportStatus()
        {
          current_block: 2328,
          direction: 1,
          ending_block: 1613030,
          listening_mode: true,
          starting_block: 1000,
	      verify: false
        }


	ethbot.exportBlockRange(starting_block,ending_block)
		starting_block: the block to start the export process from (0 to MAX_INT)
		ending_block: the block to stop the export process at (0 to MAX_INT)
        Returns: boolean
            true: export process has finished successfully
            false: export process failed at some block

	ethbot.verifySQLdata(mode,starting_block,ending_block)
        mode: 0 - use blockchain data as gold standard
              1 - use SQL data as gold standard
        starting_block: the block to start the verification process from
        ending_block: the block to end the verification process at

		Returns: boolean
            true: the verification process was successful, data does match
            false: the verification process has failed (data does't match) with the errors being reported at `status` object

	ethbot.verificationStatus()
		Returns: object
            cancelled_by_user: user has requested verifcation process to abort
            current_block_num: the current block that is being verified at this time
            error_str: the error description if the verification process has failed
            failed: true if the verification process has failed
            in_progress: true if the verification process is running
            total_threads: number of threads (parallel SQL queries) that are working at the same time to verify data
            valtr_id: the `id` of the value_transfer where data mistmatch was detected
        Example:
        > ethbot.verificationStatus()
        {
          cancelled_by_user: false,
          current_block_num: 2001,
          error_str: "",
          failed: false,
          finished_threads: 0,
          in_progress: false,
          total_threads: 89,
          valtr_id: 0
        }
        > 

	ethbot.stopVerification()
		Returns: boolean
            true: a request to stop verification process was accepted
            false: a request to stop verification process has failed

	ethbot.verifyAccount(account_address,block_num)
            account_address: 40 character long account address (without the 0x prepended)
            block_num: the block number to stop at (being the starting block always 0)
		Returns: boolean
            true: verification process has succeeded, data does match
            false: verification proces has failed, data in the blockchain isn't matching the data in the SQL database

	ethbot.verifyAllAccounts(block_num)
        block_num: block number to stop at (being the starting block always 0)
        Returns: boolean
            true: the balances of all accounts at [block_num] match thebalances of all accounts in SQL database
            false: the verification has failed (data doesn't match) with the error being reported at verification status object


#### Disabling export process at startup
Sometimes you don't want to start the export process automatically at startup, to disable it run EthBot with this flag:

	./ethbot --noexport

#### Manual export

If you want to start exporting process manually , type this on the console:

	geth> ethbot.startBlockchainExportStart(0,-1)

This will begin exporting blockchain data from the Genesis block up to the last block and then the process will enter into listening state and listen for new blocks. If you don't want EthBot to listen for new blocks replace -1 by the last block you want to export.


If you want to continue export process from the last block that have been processed by EthBot in past executions use the command with the following parameters:

	geth> ethbot.blockchainExportStart(-1,-1)

EthBot saves the last block it has processed in its own database , and the -1 value  in starting block parameter indicates EthBot that it has to continue from the point where last export process was stopped.


You can use `lastblock` command to see the value of latest processed block


	./lastblock /home/user/.ethereum/geth/ethbot_db

Or, if you are using a custom --datadir, specify it like this:

	./lastblock [datadir]/geth/ethbot_db

#### Exporting only part of the blockchain

If you only need to export a certain range of blocks without being able to enter into listening mode and without saving `last_block` variable, you may want to use the 'range of block' export process. Or you may use it if your main exporting process is already running and you don't want to turn it off, in that case  exportBlockRange() will run in parallel to the main process.

	geth> ethbot.exportBlockRange(starting_block,ending_block)

Please note that for this function to work properly, the blockchain data has to be downloaded BEFORE you use it.

#### Checking export status

To check the status of exporting process execute this on the console

	geth> ethbot.blockchainExportStatus()
	
The output will be similar to this one:

    > ethbot.blockchainExportStatus()
    {
      current_block: 2328,
      direction: 1,
      ending_block: 1613030,
      listening_mode: true,
      starting_block: 1000,
	  verify: false
    }

Which means that your process started at block 1000 , currently is exporting block 2328 and will never exit since the `listening_mode` is set to `true`, wich means EthBot will gather incoming blocks as they arrive and export them to SQL database. The ending block is 1613030 wich means you have synchronized your node up to the block number 1613030. Verification flag 'verify' is set to false wich means exported data will not be verified after exporting each block.

## Verifying data

EthBot provides mechanisms to verify the data in the SQL against blockchain and viceversa. A connection loss, data corruption or any other de-syncrhonization event may leave your SQL database in an invalid state . The verification process makes sure that the data in SQL matches the data in the blockchain. Only balances are verified at this time. Using verification methods you can identify which blocks mismatch and synchronize only those blocks, without exporting the whole blockchain again.
Normally you wouldn't want to verify every inserted block, but ocasionally you may run verification process just to be sure everything is ok.

Verification is done using Javascript console , the following functions can be used:

  * `ethbot.verifySQLdata(mode,starting_block,ending_block)`
  * `ethbot.verificationStatus()`
  * `ethbot.stopVerification()`
  * `ethbot.verifyAccount(account_address,block_num)`
  * `ethbot.verifyAllAccounts(block_num)`

#### ethbot.verifySQLdata(mode,starting_block,ending_block)

Parameters:

	mode: 0 o 1
	starting_block: the block to start from
	ending_block: the block to end at

EthBot has two verification modes

	"0" - Use LevelDB as gold standard. 

This mode of verification is enough to be sure that the SQL database is synchronized correctly

	"1" - Use Postgres SQL database as gold standard. 

If you want to double check that there are no more records in SQL DB than in the blockchain Level DB you could additionaly run this (mode=1) verification process
			

	starting_block
	ending_block

These parameters indicate the interval of blocks you want to verify. Use 0 in `starting_block` to specify genesis block, and 0 in `ending_block` to specify the last block of the blockchain.  By using "-1" in `starting_block` EthBot will start verification process from the current last block in the blockchain. By using "-1" in `ending_block` parameter EthBot will enter into listening mode and will verify all incoming blocks.

###### Examples

  1. Verify blocks upward

	ethbot.verifySQLdata(0,0,100)

Will verify in mode=0 (blockchain as gold standard) from the genesis block to block 100 and exit

  2. Verifu blocks downward

	ethbot.verifySQLdata(0,1000,100)

Will verify in mode=0 from the block 1000 to block 100, by icrementing the block counter down, ending at block 100.

  3. Verify single block
 
	ethbot.verifySQLdata(0,540,540)

Will verify a single block, the block number `540` in our case

  4. Verify full blockchain and keep verifying

	ethbot.verifySQLdata(0,0,-1)

Will start verification process in mode=0 from the genesis block , process all the blocks until the current last block, and after that will enter into listening mode and verify all incoming blocks

  5. Verify incoming (new) blocks

	ethbot.verifySQLdata(0,-1,-1)

Will start verification process in mode=0 from the last block in the blockchain , process it and then it will enter into listening mode and verify all incoming blocks

Will start verifying from the current last block 

#### ethbot.verifyAccount(account_address,block_num)

This is a different verification method. Instead of verifying an entire state, it will verify an entire account, starting from the GENESIS block up to the block specified at the parameter. If you want to be sure the SQL data of a particular account matches blockchain database this is the method to use. The process also verifies that the balance of the previous state matches the next state.

Pass -1 to `block_num` parameter if you want to verify up to the latest block in the blockchain database

#### ethbot.verifyAllAccounts(block_num)

This verification method will use ethbot.verifyAccount() for all the accounts registered at the blockchain , up to the block number `block_num`


## `last_block` parameter

	EthBot stores the parameter of the last imported block in its own
	database called `ethbot_db` wich is located under the --datadir

	To reset this parameter run this command:

		./last_block [datadir]/geth/ethbot_db [block_number]

	The number you put will be uses as starting block to export blockchain data to SQL, inclusive. 

## PLSQL Functions

EthBot provides auxiliary functions to query SQL database so you can avoid writing complex queries

#### Get account balance

	plsql> SELECT get_balance(account_id,block_num) AS balance;

This function will query the latest balance for the **account_id**. Not that it queries per block, so only using the latest block will give you the latest balance. Use -1 to specify the latest block. For example:

	psql> SELECT get_balance4block(36313,-1)

#### Get acount's transactions

	plsql> SELECT get_TXs(account_id,starting_block,ending_block)

This function will return all the transactions for an account for a range of blocks
Use -1 in `ending_block` to specify the latest block available

#### Get account's value transfers

Value transfers are transfers of Ether. The difference is that value transfer can ooccur without a transaction, like for example during mining a block. Value transfers compose a transaction , so if you need low level exploring (transfers between contracts), this function will help you.

	psql> SELECT get_VTs(account_id,starting_block,ending_block)

Use -1 in `ending_block` to specify the latest block available

## SQL query examples

  * Get transactions for an account (ToDo)



  * Get value transfers for an account (ToDo)


  * Get miner's profits (ToDo)


  * Get accounts with the highest balance (ToDo)



## Design

The SQL database contains 5 tables:

	account
	block
	uncle
	transaction
	value_transfer
	
StateDB is stored in `value_transfer` table. You can get a state for any block with this query:

	ToDo

You can find the info on relations in init_database.sql file


