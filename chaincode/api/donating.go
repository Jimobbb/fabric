package api

import (
	"chaincode/model"
	"chaincode/pkg/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// CreateDonating 发起捐赠
func CreateDonating(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 验证参数
	if len(args) != 3 {
		return shim.Error("Number of parameters less than 3")
	}
	objectOfDonating := args[0]
	donor := args[1]
	grantee := args[2]
	if objectOfDonating == "" || donor == "" || grantee == "" {
		return shim.Error("Null value exists for the parameter")
	}
	if donor == grantee {
		return shim.Error("Donor and donee cannot be the same person")
	}
	//判断objectOfDonating是否属于donor
	resultsRealEstate, err := utils.GetStateByPartialCompositeKeys2(stub, model.RealEstateKey, []string{donor, objectOfDonating})
	if err != nil || len(resultsRealEstate) != 1 {
		return shim.Error(fmt.Sprintf("Failed to verify %s belongs to %s: %s", objectOfDonating, donor, err))
	}
	var realEstate model.RealEstate
	if err = json.Unmarshal(resultsRealEstate[0], &realEstate); err != nil {
		return shim.Error(fmt.Sprintf("CreateDonating-errorinUnmarshal: %s", err))
	}
	//根据grantee获取受赠人信息
	resultsAccount, err := utils.GetStateByPartialCompositeKeys(stub, model.AccountKey, []string{grantee})
	if err != nil || len(resultsAccount) != 1 {
		return shim.Error(fmt.Sprintf("grantee failed to validate grantee information%s", err))
	}
	var accountGrantee model.Account
	//将查询到的账户信息反序列化为 accountGrantee 变量
	if err = json.Unmarshal(resultsAccount[0], &accountGrantee); err != nil {
		return shim.Error(fmt.Sprintf("Querying operator information-errorinUnmarshal: %s", err))
	}
	if accountGrantee.UserName == "manager" {
		return shim.Error(fmt.Sprintf("Cannot donate to manager%s", err))
	}
	//判断记录是否已存在，不能重复发起捐赠
	//若Encumbrance为true即说明此房产已经正在担保状态
	if realEstate.Encumbrance {
		return shim.Error("This real estate has been placed in a secured status and no further donations can be initiated.")
	}
	createTime, _ := stub.GetTxTimestamp()
	//创建一个 model.Donating 类型的变量存储捐赠相关的信息
	donating := &model.Donating{
		ObjectOfDonating: objectOfDonating,
		Donor:            donor,
		Grantee:          grantee,
		CreateTime:       time.Unix(int64(createTime.GetSeconds()), int64(createTime.GetNanos())).Local().Format("2006-01-02 15:04:05"),
		DonatingStatus:   model.DonatingStatusConstant()["donatingStart"],
	}
	// 写入账本
	if err := utils.WriteLedger(donating, stub, model.DonatingKey, []string{donating.Donor, donating.ObjectOfDonating, donating.Grantee}); err != nil {
		return shim.Error(fmt.Sprintf("%s", err))
	}
	//将房子状态设置为正在担保状态
	realEstate.Encumbrance = true
	if err := utils.WriteLedger(realEstate, stub, model.RealEstateKey, []string{realEstate.Proprietor, realEstate.RealEstateID}); err != nil {
		return shim.Error(fmt.Sprintf("%s", err))
	}
	//将本次购买交易写入账本,可供受赠人查询
	donatingGrantee := &model.DonatingGrantee{
		Grantee:    grantee,
		CreateTime: time.Unix(int64(createTime.GetSeconds()), int64(createTime.GetNanos())).Local().Format("2006-01-02 15:04:05"),
		Donating:   *donating,
	}
	if err := utils.WriteLedger(donatingGrantee, stub, model.DonatingGranteeKey, []string{donatingGrantee.Grantee, donatingGrantee.CreateTime}); err != nil {
		return shim.Error(fmt.Sprintf("Failed to write this donation transaction to the ledger%s", err))
	}
	donatingGranteeByte, err := json.Marshal(donatingGrantee)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error serializing successfully created message: %s", err))
	}
	// 成功返回
	return shim.Success(donatingGranteeByte)
}

// QueryDonatingList 查询捐赠列表
func QueryDonatingList(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var donatingList []model.Donating
	results, err := utils.GetStateByPartialCompositeKeys2(stub, model.DonatingKey, args)
	if err != nil {
		return shim.Error(fmt.Sprintf("%s", err))
	}
	for _, v := range results {
		if v != nil {
			var donating model.Donating
			err := json.Unmarshal(v, &donating)
			if err != nil {
				return shim.Error(fmt.Sprintf("QueryDonatingList-errorinUnmarshal: %s", err))
			}
			//将解析后的捐赠记录添加到 donatingList 切片中
			donatingList = append(donatingList, donating)
		}
	}
	donatingListByte, err := json.Marshal(donatingList)
	if err != nil {
		return shim.Error(fmt.Sprintf("QueryDonatingList-errorinUnmarshal: %s", err))
	}
	return shim.Success(donatingListByte)
}

// QueryDonatingListByGrantee 根据受赠人查询捐赠
func QueryDonatingListByGrantee(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error(fmt.Sprintf("Must specify grantee AccountId query"))
	}
	var donatingGranteeList []model.DonatingGrantee
	results, err := utils.GetStateByPartialCompositeKeys2(stub, model.DonatingGranteeKey, args)
	if err != nil {
		return shim.Error(fmt.Sprintf("%s", err))
	}
	for _, v := range results {
		if v != nil {
			var donatingGrantee model.DonatingGrantee
			err := json.Unmarshal(v, &donatingGrantee)
			if err != nil {
				return shim.Error(fmt.Sprintf("QueryDonatingListByGrantee-errorinUnmarshal: %s", err))
			}
			donatingGranteeList = append(donatingGranteeList, donatingGrantee)
		}
	}
	donatingGranteeListByte, err := json.Marshal(donatingGranteeList)
	if err != nil {
		return shim.Error(fmt.Sprintf("QueryDonatingListByGrantee-errorinUnmarshal: %s", err))
	}
	return shim.Success(donatingGranteeListByte)
}

// UpdateDonating 更新捐赠状态（确认受赠、取消）
func UpdateDonating(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 验证参数
	if len(args) != 4 {
		return shim.Error("Number of parameters less than 4")
	}
	objectOfDonating := args[0]
	donor := args[1]
	grantee := args[2]
	status := args[3]
	if objectOfDonating == "" || donor == "" || grantee == "" || status == "" {
		return shim.Error("Null value exists for the parameter")
	}
	if donor == grantee {
		return shim.Error("Donor and donee cannot be the same person")
	}
	//根据objectOfDonating和donor获取想要购买的房产信息，确认存在该房产
	resultsRealEstate, err := utils.GetStateByPartialCompositeKeys2(stub, model.RealEstateKey, []string{donor, objectOfDonating})
	if err != nil || len(resultsRealEstate) != 1 {
		return shim.Error(fmt.Sprintf("Failed to get information about the property you want to buy based on %s and %s: %s", objectOfDonating, donor, err))
	}
	var realEstate model.RealEstate
	if err = json.Unmarshal(resultsRealEstate[0], &realEstate); err != nil {
		return shim.Error(fmt.Sprintf("UpdateDonating-errorinUnmarshal: %s", err))
	}
	//根据grantee获取受赠人
	resultsGranteeAccount, err := utils.GetStateByPartialCompositeKeys(stub, model.AccountKey, []string{grantee})
	if err != nil || len(resultsGranteeAccount) != 1 {
		return shim.Error(fmt.Sprintf("grantee failed to validate grantee information%s", err))
	}
	var accountGrantee model.Account
	if err = json.Unmarshal(resultsGranteeAccount[0], &accountGrantee); err != nil {
		return shim.Error(fmt.Sprintf("Query grantee information-errorinUnmarshal: %s", err))
	}
	//根据objectOfDonating和donor和grantee获取捐赠信息
	resultsDonating, err := utils.GetStateByPartialCompositeKeys2(stub, model.DonatingKey, []string{donor, objectOfDonating, grantee})
	if err != nil || len(resultsDonating) != 1 {
		return shim.Error(fmt.Sprintf("Failed to get sales information based on %s and %s and %s: %s", objectOfDonating, donor, grantee, err))
	}
	var donating model.Donating
	if err = json.Unmarshal(resultsDonating[0], &donating); err != nil {
		return shim.Error(fmt.Sprintf("UpdateDonating-errorinUnmarshal: %s", err))
	}
	//不管完成还是取消操作,必须确保捐赠处于捐赠中状态
	if donating.DonatingStatus != model.DonatingStatusConstant()["donatingStart"] {
		return shim.Error("This transaction is not in donation, confirm/cancel donation failed")
	}
	//根据grantee获取买家购买信息donatingGrantee
	var donatingGrantee model.DonatingGrantee
	resultsDonatingGrantee, err := utils.GetStateByPartialCompositeKeys2(stub, model.DonatingGranteeKey, []string{grantee})
	if err != nil || len(resultsDonatingGrantee) == 0 {
		return shim.Error(fmt.Sprintf("Failed to get grantee information based on %s: %s", grantee, err))
	}
	for _, v := range resultsDonatingGrantee {
		if v != nil {
			var s model.DonatingGrantee
			err := json.Unmarshal(v, &s)
			if err != nil {
				return shim.Error(fmt.Sprintf("UpdateDonating-errorinUnmarshal: %s", err))
			}
			//检查当前买家购买信息是否满足指定的条件，即 objectOfDonating、donor 和 grantee 必须匹配。
			if s.Donating.ObjectOfDonating == objectOfDonating && s.Donating.Donor == donor && s.Grantee == grantee {
				//还必须判断状态必须为交付中,防止房子已经交易过，只是被取消了
				if s.Donating.DonatingStatus == model.DonatingStatusConstant()["donatingStart"] {
					donatingGrantee = s
					break
				}
			}
		}
	}
	var data []byte
	//判断捐赠状态
	switch status {
	case "done":
		//将房产信息转入受赠人，并重置担保状态
		realEstate.Proprietor = grantee
		realEstate.Encumbrance = false
		//realEstate.RealEstateID = stub.GetTxID() //重新更新房产ID
		if err := utils.WriteLedger(realEstate, stub, model.RealEstateKey, []string{realEstate.Proprietor, realEstate.RealEstateID}); err != nil {
			return shim.Error(fmt.Sprintf("%s", err))
		}
		//清除原来的房产信息
		if err := utils.DelLedger(stub, model.RealEstateKey, []string{donor, objectOfDonating}); err != nil {
			return shim.Error(fmt.Sprintf("%s", err))
		}
		//捐赠状态设置为完成，写入账本
		donating.DonatingStatus = model.DonatingStatusConstant()["done"]
		donating.ObjectOfDonating = realEstate.RealEstateID //重新更新房产ID
		if err := utils.WriteLedger(donating, stub, model.DonatingKey, []string{donating.Donor, objectOfDonating, grantee}); err != nil {
			return shim.Error(fmt.Sprintf("%s", err))
		}
		donatingGrantee.Donating = donating
		if err := utils.WriteLedger(donatingGrantee, stub, model.DonatingGranteeKey, []string{donatingGrantee.Grantee, donatingGrantee.CreateTime}); err != nil {
			return shim.Error(fmt.Sprintf("Failed to write this donation transaction to the ledger%s", err))
		}
		data, err = json.Marshal(donatingGrantee)
		if err != nil {
			return shim.Error(fmt.Sprintf("Error in information about serialized donation transactions: %s", err))
		}
		break
	case "cancelled":
		//重置房产信息担保状态
		realEstate.Encumbrance = false
		if err := utils.WriteLedger(realEstate, stub, model.RealEstateKey, []string{realEstate.Proprietor, realEstate.RealEstateID}); err != nil {
			return shim.Error(fmt.Sprintf("%s", err))
		}
		//更新捐赠状态
		donating.DonatingStatus = model.DonatingStatusConstant()["cancelled"]
		if err := utils.WriteLedger(donating, stub, model.DonatingKey, []string{donating.Donor, donating.ObjectOfDonating, donating.Grantee}); err != nil {
			return shim.Error(fmt.Sprintf("%s", err))
		}
		donatingGrantee.Donating = donating
		if err := utils.WriteLedger(donatingGrantee, stub, model.DonatingGranteeKey, []string{donatingGrantee.Grantee, donatingGrantee.CreateTime}); err != nil {
			return shim.Error(fmt.Sprintf("%s", err))
		}
		data, err = json.Marshal(donatingGrantee)
		if err != nil {
			return shim.Error(fmt.Sprintf("%s", err))
		}
		break
	default:
		return shim.Error(fmt.Sprintf("%sStatus is error", status))
	}
	return shim.Success(data)
}
