/*
 * Copyright 2018 The OpenWallet Authors
 * This file is part of the OpenWallet library.
 *
 * The OpenWallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The OpenWallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package tezos

import (
	"fmt"
	"github.com/blocktree/OpenWallet/common"
	"github.com/blocktree/OpenWallet/common/file"
	"github.com/blocktree/OpenWallet/console"
	"github.com/blocktree/OpenWallet/timer"
	//"github.com/shopspring/decimal"
	"log"
	"path/filepath"
	"strings"
	"errors"
	"github.com/shopspring/decimal"
	"strconv"
)


type WalletManager struct{}

//初始化配置流程
func (w *WalletManager) InitConfigFlow() error {

	var (
		err        error
		apiURL     string
		walletPath string
		//汇总阀值
		threshold string
		//最小转账额度
		minSendAmount string
		//最小矿工费
		minFees string
		//gas 费用上限
		gasLimit string
		//存储费用上限，一般为0.预留这个
		storageLimit string
		//汇总地址
		sumAddress string
		filePath   string
	)

	for {

		fmt.Printf("[Start setup wallet config]\n")

		apiURL, err = console.InputText("Set node API url: ", true)
		if err != nil {
			return err
		}


		sumAddress, err = console.InputText("Set summary address: ", false)
		if err != nil {
			return err
		}


		threshold, err = console.InputRealNumber("Set summary threshold: ", true)
		if err != nil {
			return err
		}

		minSendAmount, err = console.InputRealNumber("Set minimum transfer amount: ", true)
		if err != nil {
			return err
		}

		fmt.Printf("[Suggest the transfer fees no less than %f]\n", 0.0001)
		minFees, err = console.InputRealNumber("Set transfer fees: ", true)
		if err != nil {
			return err
		}

		fmt.Printf("[Suggest the gas limit no less than %f]\n", 0.0001)
		gasLimit, err = console.InputRealNumber("Set gas limit: ", true)
		if err != nil {
			return err
		}

		fmt.Printf("[Suggest the storage limit %f]\n", 0)
		storageLimit, err = console.InputRealNumber("Set storage limit: ", true)
		if err != nil {
			return err
		}

		//最小发送数量不能超过汇总阀值
		if minFees > minSendAmount {
			if minSendAmount > threshold {
				return errors.New("The summary threshold must be greater than the minimum transfer amount! ")
			}

			return errors.New("The minimum transfer amount must be greater than the transfer fees! ")
		}

		//换两行
		fmt.Println()
		fmt.Println()

		//打印输入内容
		fmt.Printf("Please check the following setups is correct?\n")
		fmt.Printf("-----------------------------------------------------------\n")
		fmt.Printf("Node API url: %s\n", apiURL)
		fmt.Printf("Wallet main net filePath: %s\n", walletPath)
		fmt.Printf("Summary address: %s\n", sumAddress)
		fmt.Printf("Summary threshold: %s\n", threshold)
		fmt.Printf("Minimum transfer amount: %s\n", minSendAmount)
		fmt.Printf("Transfer fees: %s\n", minFees)
		fmt.Printf("Gas limit: %s\n", gasLimit)
		fmt.Printf("Storage limit fees: %s\n", storageLimit)
		fmt.Printf("-----------------------------------------------------------\n")

		flag, err := console.Stdin.PromptConfirm("Confirm to save the setups?")
		if err != nil {
			return err
		}

		if !flag {
			continue
		} else {
			break
		}

	}

	//换两行
	fmt.Println()
	fmt.Println()

	_, filePath, err = newConfigFile(apiURL, sumAddress, threshold, minSendAmount, minFees, gasLimit, storageLimit)

	fmt.Printf("Config file create, file path: %s\n", filePath)

	return nil

}

//查看配置信息
func (w *WalletManager) ShowConfig() error {
	return printConfig()
}

//创建钱包流程
func (w *WalletManager) CreateWalletFlow() error {
	var (
		name     string
		err      error
	)

	//先加载是否有配置文件
	err = loadConfig()
	if err != nil {
		return err
	}

	// 等待用户输入钱包名字
	name, err = console.InputText("Enter wallet's name: ", true)

	// 随机生成密钥
	return CreateNewWallet(name)
}

//创建地址流程
func (w *WalletManager) CreateAddressFlow() error {
	//先加载是否有配置文件
	err := loadConfig()
	if err != nil {
		return err
	}

	//查询所有钱包信息
	wallets, err := GetWallets()
	if err != nil {
		fmt.Printf("The node did not create any wallet!\n")
		return err
	}

	//打印钱包
	printWalletList(wallets)

	fmt.Printf("[Please select a wallet account to create address] \n")

	//选择钱包
	num, err := console.InputNumber("Enter wallet number: ", true)
	if err != nil {
		return err
	}

	if int(num) >= len(wallets) {
		return errors.New("Input number is out of index! ")
	}

	wallet := wallets[num]

	// 输入地址数量
	count, err := console.InputNumber("Enter the number of addresses you want: ", false)
	if err != nil {
		return err
	}

	if count > maxAddresNum {
		return errors.New(fmt.Sprintf("The number of addresses can not exceed %d\n", maxAddresNum))
	}

	//输入密码
	password, err := console.InputPassword(false, 8)

	log.Printf("Start batch creation\n")
	log.Printf("-------------------------------------------------\n")

	filePath, _, err := CreateBatchAddress(wallet.WalletID, password, count)
	if err != nil {
		return err
	}

	log.Printf("-------------------------------------------------\n")
	log.Printf("All addresses have created, file path:%s\n", filePath)

	return nil
}

//汇总钱包流程

/*

汇总执行流程：
1. 执行启动汇总某个币种命令。
2. 列出该币种的全部可用钱包信息。
3. 输入需要汇总的钱包序号数组（以,号分隔）。
4. 输入每个汇总钱包的密码，完成汇总登记。
5. 工具启动定时器监听钱包，并输出日志到log文件夹。
6. 待已登记的汇总钱包达到阀值，发起账户汇总到配置下的地址。

*/

// SummaryFollow 汇总流程
func (w *WalletManager) SummaryFollow() error {

	var (
		endRunning = make(chan bool, 1)
	)

	//先加载是否有配置文件
	err := loadConfig()
	if err != nil {
		return err
	}

	//判断汇总地址是否存在
	if len(sumAddress) == 0 {

		return errors.New(fmt.Sprintf("Summary address is not set. Please set it in './conf/%s.ini' \n", Symbol))
	}

	//查询所有钱包信息
	wallets, err := GetWallets()
	if err != nil {
		fmt.Printf("The node did not create any wallet!\n")
		return err
	}

	//打印钱包
	printWalletList(wallets)

	fmt.Printf("[Please select the wallet to summary, and enter the numbers split by ','." +
		" For example: 0,1,2,3] \n")

	// 等待用户输入钱包名字
	nums, err := console.InputText("Enter the No. group: ", true)
	if err != nil {
		return err
	}

	//分隔数组
	wallet_array := strings.Split(nums, ",")

	for _, numIput := range wallet_array {
		if common.IsNumberString(numIput) {
			numInt := common.NewString(numIput).Int()
			if numInt < len(wallets) {
				w := wallets[numInt]

				fmt.Printf("Register summary wallet [%s]-[%s]\n", w.Alias, w.WalletID)
				//输入钱包密码完成登记
				password, err := console.InputPassword(false, 8)
				if err != nil {
					return err
				}

				w.Password = password

				AddWalletInSummary(w.WalletID, w)
			} else {
				return errors.New("The input No. out of index! ")
			}
		} else {
			return errors.New("The input No. is not numeric! ")
		}
	}

	if len(walletsInSum) == 0 {
		return errors.New("Not summary wallets to register! ")
	}

	fmt.Printf("The timer for summary has started. Execute by every %v seconds.\n", cycleSeconds.Seconds())

	//启动钱包汇总程序
	sumTimer := timer.NewTask(cycleSeconds, SummaryWallets)
	sumTimer.Start()

	<-endRunning

	return nil
}

//备份钱包流程
func (w *WalletManager) BackupWalletFlow() error {

	var (
		err error
	)

	//先加载是否有配置文件
	err = loadConfig()
	if err != nil {
		return err
	}

	//建立备份路径
	backupPath := filepath.Join(keyDir, "backup")
	file.MkdirAll(backupPath)


	err = file.Copy(dbPath, filepath.Join(backupPath))
	if err != nil {
		return err
	}

	//输出备份导出目录
	log.Printf("Wallet backup file path: %s", backupPath)

	return nil

}

//SendTXFlow 发送交易
func (w *WalletManager) TransferFlow() error {
	//先加载是否有配置文件
	err := loadConfig()
	if err != nil {
		return err
	}

	wallets, err := GetWallets()
	if err != nil {
		return err
	}

	//打印钱包列表
	printWalletList(wallets)

	fmt.Printf("[Please select a wallet to send transaction] \n")

	//选择钱包
	num, err := console.InputNumber("Enter wallet No. : ", true)
	if err != nil {
		return err
	}

	if int(num) >= len(wallets) {
		return errors.New("Input number is out of index! ")
	}

	wallet := wallets[num]

	// 等待用户输入发送数量
	amount, err := console.InputRealNumber("Enter amount to send: ", true)
	if err != nil {
		return err
	}

	atculAmount, _ := decimal.NewFromString(amount)
	atculAmount = atculAmount.Mul(coinDecimal)

	// 等待用户输入发送地址
	receiver, err := console.InputText("Enter receiver address: ", true)
	if err != nil {
		return err
	}

	//输入密码解锁钱包
	password, err := console.InputPassword(false, 8)
	if err != nil {
		return err
	}
	//加载钱包数据库数据
	db, err := wallet.OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()

	var keys []*Key
	db.All(&keys)

	haveEnoughBalance := false


	type sendStruct struct {
		sednKeys *Key
		fee      decimal.Decimal
		amount   decimal.Decimal
	}

	var sends []sendStruct
	var resultSub decimal.Decimal = atculAmount
	//新建发送地址列表以及验证余额是否足够
	for _, k := range keys {
		//get balance
		decimal_balance, _ := decimal.NewFromString(string(callGetbalance(k.Address)))

		//判断是否是reveal交易
		fee := minFees
		isReverl := isReverlKey(k.Address)
		if isReverl {
			//多了reveal操作后，fee * 2
			fee = minFees.Mul(decimal.RequireFromString("2"))
		}
		// 将该地址多余额减去矿工费
		amount := decimal_balance.Sub(fee)
		//该地址预留一点币，否则交易会失败，暂定0.00002 tez
		amount = amount.Sub(decimal.RequireFromString("20"))
		if amount.IntPart() < 0 {
			continue
		}

		if resultSub.LessThanOrEqual(amount) {
			send := sendStruct{k, fee, resultSub}
			sends = append(sends, send)
			haveEnoughBalance = true
			break
		} else {
			send := sendStruct{k, fee, amount}
			sends = append(sends, send)
			log.Printf("address:%s, amount:%d, resultSub:%d\n", k.Address, amount.IntPart(), resultSub.IntPart())
		}
		resultSub = resultSub.Sub(amount)
		log.Printf("resultSub:%d\n", resultSub.IntPart())
	}

	if haveEnoughBalance {
		for _, send := range sends {
			sk, _ := Decrypt(password, send.sednKeys.PrivateKey)
			send.sednKeys.PrivateKey = sk

			txid, _ := transfer(*send.sednKeys, receiver, strconv.FormatInt(minFees.IntPart(), 10), strconv.FormatInt(gasLimit.IntPart(), 10),
				strconv.FormatInt(storageLimit.IntPart(), 10),strconv.FormatInt(send.amount.IntPart(), 10))
			log.Printf("transfer address:%s, to address:%s, amount:%d, txid:%s\n", send.sednKeys.Address, sumAddress, send.amount.IntPart(), txid)
		}
	} else {
		log.Printf("not enough balance\n")
		return errors.New("Wallet have not enough balance to transfer\n")
	}

	return nil
}

//GetWalletList 获取钱包列表
func (w *WalletManager) GetWalletList() error {

	var (
		err error
	)

	//先加载是否有配置文件
	err = loadConfig()
	if err != nil {
		return err
	}

	list, err := GetWallets()
	if err != nil {
		return err
	}

	//打印钱包列表
	printWalletList(list)

	return nil
}

//RestoreWalletFlow 恢复钱包
func (w *WalletManager) RestoreWalletFlow() error {

	fmt.Printf("Restore wallet is unavailable now.\n")

	return nil
}


//SetConfigFlow 初始化配置流程
func (w *WalletManager) SetConfigFlow(subCmd string) error {
	file := configFilePath + configFileName
	fmt.Printf("You can run 'vim %s' to edit %s config.\n", file, subCmd)
	return nil
}

//ShowConfigInfo 查看配置信息
func (w *WalletManager) ShowConfigInfo(subCmd string) error {
	printConfig()
	return nil
}
