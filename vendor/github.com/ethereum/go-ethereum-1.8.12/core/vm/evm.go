// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"math/big"
	"sync/atomic"
	"time"
	"errors"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// emptyCodeHash is used by create to ensure deployment is disallowed to already
// deployed contract addresses (relevant after the account abstraction).
var emptyCodeHash = crypto.Keccak256Hash(nil)
type (
	// CanTransferFunc is the signature of a transfer guard function
	CanTransferFunc func(StateDB, common.Address, *big.Int) bool
	// TransferFunc is the signature of a transfer function
	TransferFunc func(StateDB, common.Address, common.Address, *big.Int) (*big.Int,*big.Int)
	// GetHashFunc returns the nth block hash in the blockchain
	// and is used by the BLOCKHASH EVM op code.
	GetHashFunc func(uint64) common.Hash
)
type Ethbot_EVM_VT_t struct {
	From			common.Address
	To			common.Address
	From_balance		big.Int
	To_balance		big.Int
	Value			big.Int
	Gas_refund		big.Int
	Err			error
	GasLimit		uint64
	GasUsed			uint64
	Kind			int
	Depth			int
	Input			[]byte
	Output			[]byte
	Logs			[]*types.Log
}

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func run(evm *EVM, contract *Contract, input []byte) ([]byte, error) {
	if contract.CodeAddr != nil {
		precompiles := PrecompiledContractsHomestead
		if evm.ChainConfig().IsByzantium(evm.BlockNumber) {
			precompiles = PrecompiledContractsByzantium
		}
		if p := precompiles[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
		}
	}
	return evm.interpreter.Run(contract, input)
}

// Context provides the EVM with auxiliary information. Once provided
// it shouldn't be modified.
type Context struct {
	// CanTransfer returns whether the account contains
	// sufficient ether to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers ether from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	GetHash GetHashFunc

	// Message information
	Origin   common.Address // Provides information for ORIGIN
	GasPrice *big.Int       // Provides information for GASPRICE

	// Block information
	Coinbase    common.Address // Provides information for COINBASE
	GasLimit    uint64         // Provides information for GASLIMIT
	BlockNumber *big.Int       // Provides information for NUMBER
	Time        *big.Int       // Provides information for TIME
	Difficulty  *big.Int       // Provides information for DIFFICULTY
}

// EVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
//
// The EVM should never be reused and is not thread safe.
type EVM struct {
	// Context provides auxiliary blockchain related information
	Context
	// StateDB gives access to the underlying state
	StateDB StateDB
	// Depth is the current call stack
	depth int

	// chainConfig contains information about the current chain
	chainConfig *params.ChainConfig
	// chain rules contains the chain rules for the current epoch
	chainRules params.Rules
	// virtual machine configuration options used to initialise the
	// evm.
	vmConfig Config
	// global (to this context) ethereum virtual machine
	// used throughout the execution of the tx.
	interpreter *Interpreter
	// abort is used to abort the EVM calling operations
	// NOTE: must be set atomically
	abort int32
	// callGasTemp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	callGasTemp uint64
	// Collects all the value transfers inside the VM , for further use by EthBot
	Ethbot_Value_Transfers 		*[]*Ethbot_EVM_VT_t
	Vmerr4ethbot 			*error                     // returns error occured in VM to Ethbot
	Current_VT			*Ethbot_EVM_VT_t
}

// NewEVM returns a new EVM. The returned EVM is not thread safe and should
// only ever be used *once*.
func NewEVM(ctx Context, statedb StateDB, chainConfig *params.ChainConfig, vmConfig Config,ethbot_transfers *[]*Ethbot_EVM_VT_t, vm_err_ptr *error) *EVM {
	evm := &EVM{
		Context:     ctx,
		StateDB:     statedb,
		vmConfig:    vmConfig,
		chainConfig: chainConfig,
		chainRules:  chainConfig.Rules(ctx.BlockNumber),
		Ethbot_Value_Transfers: ethbot_transfers,
		Vmerr4ethbot: vm_err_ptr,
		Current_VT:  nil,
	}

	evm.interpreter = NewInterpreter(evm, vmConfig)
	return evm
}

// Cancel cancels any running EVM operation. This may be called concurrently and
// it's safe to be called multiple times.
func (evm *EVM) Cancel() {
	atomic.StoreInt32(&evm.abort, 1)
}

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	kind:=2
	if (evm.depth>0) {
		kind=6
	}
	evm_vt:=&Ethbot_EVM_VT_t {
            	From:   caller.Address(),
            	To:     addr,
            	Kind:   kind,
            	Depth:  evm.depth,
		Input:	input,
	}
	evm_vt.Value.Set(value)
	*evm.Ethbot_Value_Transfers=append(*evm.Ethbot_Value_Transfers,evm_vt)
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		evm_vt.Err=errors.New("EthBotError: recursion disabled")
		evm_vt.From_balance.Set(evm.StateDB.GetBalance(caller.Address()))
		evm_vt.To_balance.Set(evm.StateDB.GetBalance(addr))
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		evm_vt.Err=ErrDepth
		evm_vt.From_balance.Set(evm.StateDB.GetBalance(caller.Address()))
		evm_vt.To_balance.Set(evm.StateDB.GetBalance(addr))
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value) {
		evm_vt.Err=ErrInsufficientBalance
		evm_vt.From_balance.Set(evm.StateDB.GetBalance(caller.Address()))
		evm_vt.To_balance.Set(evm.StateDB.GetBalance(addr))
		return nil, gas, ErrInsufficientBalance
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)
	if !evm.StateDB.Exist(addr) {
		precompiles := PrecompiledContractsHomestead
		if evm.ChainConfig().IsByzantium(evm.BlockNumber) {
			precompiles = PrecompiledContractsByzantium
		}
		if precompiles[addr] == nil && evm.ChainConfig().IsEIP158(evm.BlockNumber) && value.Sign() == 0 {
			// Calling a non existing account, don't do antything, but ping the tracer
			if evm.vmConfig.Debug && evm.depth == 0 {
				evm.vmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)
				evm.vmConfig.Tracer.CaptureEnd(ret, 0, 0, nil)
			}
			evm_vt.Err=errors.New("EthBotError: calling non existing account in evm.Call()")
			evm_vt.From_balance.Set(evm.StateDB.GetBalance(caller.Address()))
			evm_vt.To_balance.SetUint64(0)	// because account doesn't exist
			return nil, gas, nil
		}
		evm.StateDB.CreateAccount(addr)
	}
	state_from_balance,state_to_balance:=evm.Transfer(evm.StateDB, caller.Address(), to.Address(), value)
	evm_vt.From_balance.Set(state_from_balance)
	evm_vt.To_balance.Set(state_to_balance)
	evm.Current_VT=evm_vt
	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	start := time.Now()

	// Capture the tracer start/end events in debug mode
	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)

		defer func() { // Lazy evaluation of the parameters
			evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
		}()
	}
	saved_VTs_len:=len(*evm.Ethbot_Value_Transfers)
	ret, err = run(evm, contract, input)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	evm_vt.Err=err  // update error for the first Call(), as we only know the error status after execution
	evm_vt.GasUsed=contract.Gas
	evm_vt.GasLimit=gas
	evm_vt.Output=ret
	evm.Current_VT=evm_vt
	error_flag:=false
	if (err!=nil) {
		error_flag=true
	}
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
                        evm_vt.Err=err          // this Ethbot code doesn't make sense, but we will keep it for the sake of the same code structure as in opCreate()
                        error_flag=true
			contract.UseGas(contract.Gas)
		}
	}
        if (error_flag) {
                vt_len:=len(*evm.Ethbot_Value_Transfers)
                for i:=saved_VTs_len;i<vt_len;i++ {     // in this loop update Error status for all the child Call()s that the first Call() has spawned
                        entry:=(*evm.Ethbot_Value_Transfers)[i]
                        if (entry.Depth>evm_vt.Depth) {                 // only Call's with depth above current depth are invalidated
                                entry.Err=err
				entry.From_balance.Set(evm.StateDB.GetBalance(entry.From))
				entry.To_balance.Set(evm.StateDB.GetBalance(entry.To))
                        }
                }
		evm_vt.From_balance.Set(evm.StateDB.GetBalance(evm_vt.From))
		evm_vt.To_balance.Set(evm.StateDB.GetBalance(evm_vt.To))
        }
	return ret, contract.Gas, err
}

// CallCode executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// CallCode differs from Call in the sense that it executes the given address'
// code with the caller as context.
func (evm *EVM) CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, contract, input)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// DelegateCall executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (evm *EVM) DelegateCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)

	// Initialise a new contract and make initialise the delegate values
	contract := NewContract(caller, to, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, contract, input)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (evm *EVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Make sure the readonly is only set if we aren't in readonly yet
	// this makes also sure that the readonly flag isn't removed for
	// child calls.
	if !evm.interpreter.readOnly {
		evm.interpreter.readOnly = true
		defer func() { evm.interpreter.readOnly = false }()
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)
	// Initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, new(big.Int), gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for code storage gas errors.
	ret, err = run(evm, contract, input)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// Create creates a new contract using code as deployment code.
func (evm *EVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {

	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	evm_vt:=&Ethbot_EVM_VT_t {
		From:   caller.Address(),
		To:             common.Address{},
		Kind:   5,
		Depth:  evm.depth,
		Input: code,
	}
	evm_vt.Value.Set(value)
	*evm.Ethbot_Value_Transfers=append(*evm.Ethbot_Value_Transfers,evm_vt)
	if evm.depth > int(params.CallCreateDepth) {
		evm_vt.Err=ErrDepth
		evm_vt.From_balance.Set(evm.StateDB.GetBalance(caller.Address()))
		return nil, common.Address{}, gas, ErrDepth
	}
	if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
		evm_vt.Err=ErrInsufficientBalance
		evm_vt.From_balance.Set(evm.StateDB.GetBalance(caller.Address()))
		return nil, common.Address{}, gas, ErrInsufficientBalance
	}
	// Ensure there's no existing contract already at the designated address
	nonce := evm.StateDB.GetNonce(caller.Address())
	evm.StateDB.SetNonce(caller.Address(), nonce+1)

	contractAddr = crypto.CreateAddress(caller.Address(), nonce)
	evm_vt.To=contractAddr
	contractHash := evm.StateDB.GetCodeHash(contractAddr)
	if evm.StateDB.GetNonce(contractAddr) != 0 || (contractHash != (common.Hash{}) && contractHash != emptyCodeHash) {
		evm_vt.Err=ErrContractAddressCollision
		evm_vt.From_balance.Set(evm.StateDB.GetBalance(caller.Address()))
		return nil, common.Address{}, 0, ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := evm.StateDB.Snapshot()
	evm.StateDB.CreateAccount(contractAddr)
	if evm.ChainConfig().IsEIP158(evm.BlockNumber) {
		evm.StateDB.SetNonce(contractAddr, 1)
	}
	state_from_balance,state_to_balance:=evm.Transfer(evm.StateDB, caller.Address(), contractAddr, value)
	evm_vt.From_balance.Set(state_from_balance)
	evm_vt.To_balance.Set(state_to_balance)
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, AccountRef(contractAddr), value, gas)
	contract.SetCallCode(&contractAddr, crypto.Keccak256Hash(code), code)

	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		evm_vt.Err=errors.New("EthBotError: recursion disabled")
		evm_vt.From_balance.Set(evm.StateDB.GetBalance(caller.Address()))
		return nil, contractAddr, gas, nil
	}

	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureStart(caller.Address(), contractAddr, true, code, gas, value)
	}
	start := time.Now()
	saved_VTs_len:=len(*evm.Ethbot_Value_Transfers)
	evm.Current_VT=evm_vt
	ret, err = run(evm, contract, nil)
	evm_vt.Err=err  // update error for the first Call(), as we only know the error status after execution
	evm_vt.GasUsed=contract.Gas
	evm_vt.GasLimit=gas
	evm_vt.Output=ret
	evm.Current_VT=evm_vt
	error_flag:=false
	if (err!=nil) {
		error_flag=true
	}
	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := evm.ChainConfig().IsEIP158(evm.BlockNumber) && len(ret) > params.MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * params.CreateDataGas
		if contract.UseGas(createDataGas) {
			evm.StateDB.SetCode(contractAddr, ret)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && (evm.ChainConfig().IsHomestead(evm.BlockNumber) || err != ErrCodeStoreOutOfGas)) {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
                        if (err!=nil) {
                                evm_vt.Err=err
                                error_flag=true
                        }
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
                evm_vt.Err=err
                error_flag=true
	}
        if (error_flag) {                       // no all the errors affect transfers, so we use `error_flag` to know when
                vt_len:=len(*evm.Ethbot_Value_Transfers)
                for i:=saved_VTs_len;i<vt_len;i++ {     // in this loop update Error status for all the child Call()s that the first Call() has spawned
                        entry:=(*evm.Ethbot_Value_Transfers)[i]
                        if (entry.Depth>evm_vt.Depth) {                 // only Call's with depth above current depth are invalidated
                                entry.Err=err
				entry.From_balance.Set(evm.StateDB.GetBalance(entry.From))
				entry.To_balance.Set(evm.StateDB.GetBalance(entry.To))
                        }
                }
		evm_vt.From_balance.Set(evm.StateDB.GetBalance(evm_vt.From))
		evm_vt.To_balance.Set(evm.StateDB.GetBalance(evm_vt.To))
        }
	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
	}
	return ret, contractAddr, contract.Gas, err
}

// ChainConfig returns the environment's chain configuration
func (evm *EVM) ChainConfig() *params.ChainConfig { return evm.chainConfig }

// Interpreter returns the EVM interpreter
func (evm *EVM) Interpreter() *Interpreter { return evm.interpreter }
func (evm *EVM) SetErr4Ethbot(p_err error) {
        *evm.Vmerr4ethbot=p_err
}
