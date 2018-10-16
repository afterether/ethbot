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
)
func (ebapi *EthBotAPI) Verificationstatus() Verification_t {
	return ethbot_instance.verification
}
func (ebapi *EthBotAPI) Blockchainexportstatus() Export_t {
	return ethbot_instance.export
}
func (ebapi *EthBotAPI) Tokenexportstatus() TokenExport_t {
	return ethbot_instance.token_export
}
func (ebapi *EthBotAPI) Verifysqldata1(block_num Block_num_t) bool {
	if (block_num>-1) {
		return verify_SQL_data(VERIFICATION_LEVELDB,block_num,block_num)
	} else {
		return false;
	}
}
func (ebapi *EthBotAPI) Verifysqldata2(block_num Block_num_t) bool {
	if (block_num>-1) {
		return verify_SQL_data(VERIFICATION_SQL,block_num,block_num)
	} else {
		return false;
	}
}
func (ebapi *EthBotAPI) Stopverification() {
	stop_verification()
}
func (ebapi *EthBotAPI) Verifyaccount(account_addr_str string,block_num Block_num_t) bool {
	return verify_single_account(account_addr_str,block_num)
}
func (ebapi *EthBotAPI) Verifyallaccounts(block_num Block_num_t) bool {
	return verify_all_accounts(block_num)
}
func (ebapi *EthBotAPI) Exportblockrange(starting_block Block_num_t,ending_block Block_num_t) bool {
	return ethbot_instance.export_block_range(starting_block,ending_block)
}
func (ebapi *EthBotAPI) Blockchainexportstart(starting_block Block_num_t,ending_block Block_num_t) error {
	return ethbot_instance.blockchain_export_start(starting_block,ending_block)
}
func (ebapi *EthBotAPI) Blockchainexportstop() bool {
	return ethbot_instance.blockchain_export_stop()
}
func (ebapi *EthBotAPI) Updatemainstats(block_num int) bool {
	return ethbot_instance.update_main_stats(block_num)
}
func (ebapi *EthBotAPI) Fixlastbalances() bool {
	return ethbot_instance.fix_last_balances();
}
func (ebapi *EthBotAPI) Verifylastbalances() bool {
	return ethbot_instance.verify_last_balances();
}
func (ebapi *EthBotAPI) Alarmson() bool {
	return ethbot_instance.alarms_on()
}
func (ebapi *EthBotAPI) Alarmsoff() bool {
	return ethbot_instance.alarms_off()
}
func (ebapi *EthBotAPI) Tokenexportstart(starting_block Block_num_t,ending_block Block_num_t) error {
	return ethbot_instance.token_export_start(starting_block,ending_block)
}
func (ebapi *EthBotAPI) Tokenexportstop() bool {
	return ethbot_instance.token_export_stop();
}
func (ebapi *EthBotAPI) Verifytoken(contract_addr_str string,block_num Block_num_t) bool {
	return verify_token(contract_addr_str,block_num)
}
func (ebapi *EthBotAPI) Verifyalltokens(block_num Block_num_t) bool {
	return verify_all_tokens(block_num)
}
