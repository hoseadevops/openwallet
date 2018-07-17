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

package assets

import "github.com/blocktree/OpenWallet/openwallet"

//MerchantAssets 钱包与商户交互的资产接口
type MerchantAssets interface {

	//CreateMerchantWallet 创建钱包
	CreateMerchantWallet(alias string, password string) (*openwallet.Wallet, error)

	//GetMerchantWalletList 获取钱包列表
	GetMerchantWalletList() ([]*openwallet.Wallet, error)

	//ConfigMerchantWallet 钱包工具配置接口
	ConfigMerchantWallet(wallet *openwallet.Wallet) error

	//ImportMerchantAddress 导入地址
	ImportMerchantAddress(addresses []*openwallet.Address) error

	//CreateMerchantAddress 创建钱包地址
	CreateMerchantAddress(walletID string, count int) ([]*openwallet.Address, error)

	//GetMerchantAddressList 获取钱包地址
	GetMerchantAddressList(walletID string, offset uint64, limit uint64) ([]*openwallet.Address, error)

}

// GetMerchantAssets 根据币种类型获取已注册的管理者
func GetMerchantAssets(symbol string) MerchantAssets {
	manager, ok := managers[symbol].(MerchantAssets)
	if !ok {
		return nil
	}
	return manager
}
