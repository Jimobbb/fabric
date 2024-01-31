package api

import (
	"chaincode/model"
	"chaincode/pkg/utils"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// CreateRealEstate 新建管理员
func CreateRealEstate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 验证参数
	if len(args) != 4 {
		return shim.Error("Number of parameters less than 4")
	}
	accountId := args[0] //accountId用于验证是否为管理员
	proprietor := args[1]
	totalArea := args[2]
	livingSpace := args[3]
	if accountId == "" || proprietor == "" || totalArea == "" || livingSpace == "" {
		return shim.Error("Null value exists for the parameter")
	}
	if accountId == proprietor {
		return shim.Error("The operator should be an administrator and cannot be the same as the owner.")
	}
	// 参数数据格式转换
	var formattedTotalArea float64
	//totalArea转换为float64
	if val, err := strconv.ParseFloat(totalArea, 64); err != nil {
		return shim.Error(fmt.Sprintf("Error converting totalArea parameter format: %s", err))
	} else {
		formattedTotalArea = val
	}
	var formattedLivingSpace float64
	//livingSpace转换为float64
	if val, err := strconv.ParseFloat(livingSpace, 64); err != nil {
		return shim.Error(fmt.Sprintf("livingSpace参数格式转换出错: %s", err))
	} else {
		formattedLivingSpace = val
	}
	//判断是否管理员操作
	resultsAccount, err := utils.GetStateByPartialCompositeKeys(stub, model.AccountKey, []string{accountId})
	if err != nil || len(resultsAccount) != 1 {
		return shim.Error(fmt.Sprintf("Error converting totalArea parameter format:%s", err))
	}
	var account model.Account
	if err = json.Unmarshal(resultsAccount[0], &account); err != nil {
		return shim.Error(fmt.Sprintf("errorinUnmarshal: %s", err))
	}
	if account.UserName != "manager" {
		return shim.Error(fmt.Sprintf("Insufficient operator authority%s", err))
	}
	//判断业主是否存在
	resultsProprietor, err := utils.GetStateByPartialCompositeKeys(stub, model.AccountKey, []string{proprietor})
	if err != nil || len(resultsProprietor) != 1 {
		return shim.Error(fmt.Sprintf("Owner proprietor information validation failed%s", err))
	}
	realEstate := &model.RealEstate{
		RealEstateID: stub.GetTxID()[:16],
		Proprietor:   proprietor,
		Encumbrance:  false,
		TotalArea:    formattedTotalArea,
		LivingSpace:  formattedLivingSpace,
	}
	// 写入账本
	if err := utils.WriteLedger(realEstate, stub, model.RealEstateKey, []string{realEstate.Proprietor, realEstate.RealEstateID}); err != nil {
		return shim.Error(fmt.Sprintf("%s", err))
	}
	//将成功创建的信息返回
	realEstateByte, err := json.Marshal(realEstate)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error serializing successfully created message: %s", err))
	}
	// 成功返回
	return shim.Success(realEstateByte)
}

// QueryRealEstateList 查询房地产(可查询所有，也可根据所有人查询名下房产)
func QueryRealEstateList(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var realEstateList []model.RealEstate
	results, err := utils.GetStateByPartialCompositeKeys2(stub, model.RealEstateKey, args)
	if err != nil {
		return shim.Error(fmt.Sprintf("%s", err))
	}
	for _, v := range results {
		if v != nil {
			var realEstate model.RealEstate
			err := json.Unmarshal(v, &realEstate)
			if err != nil {
				return shim.Error(fmt.Sprintf("QueryRealEstateList-errorinUnmarshal: %s", err))
			}
			realEstateList = append(realEstateList, realEstate)
		}
	}
	realEstateListByte, err := json.Marshal(realEstateList)
	if err != nil {
		return shim.Error(fmt.Sprintf("QueryRealEstateList-errorinUnmarshal: %s", err))
	}
	return shim.Success(realEstateListByte)
}
