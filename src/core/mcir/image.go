package mcir

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/xwb1989/sqlparser"

	"github.com/cloud-barista/cb-spider/interface/api"
	"github.com/cloud-barista/cb-tumblebug/src/core/common"
)

// 2020-04-03 https://github.com/cloud-barista/cb-spider/blob/master/cloud-control-manager/cloud-driver/interfaces/resources/ImageHandler.go

type SpiderImageReqInfoWrapper struct { // Spider
	ConnectionName string
	ReqInfo        SpiderImageInfo
}

/*
type SpiderImageReqInfo struct { // Spider
	//IId   IID 	// {NameId, SystemId}
	Name string
	// @todo
}
*/

type SpiderImageInfo struct { // Spider
	// Fields for request
	Name string

	// Fields for response
	IId          common.IID // {NameId, SystemId}
	GuestOS      string     // Windows7, Ubuntu etc.
	Status       string     // available, unavailable
	KeyValueList []common.KeyValue
}

type TbImageReq struct {
	Name           string `json:"name"`
	ConnectionName string `json:"connectionName"`
	CspImageId     string `json:"cspImageId"`
	Description    string `json:"description"`
}

type TbImageInfo struct {
	Id                   string            `json:"id"`
	Name                 string            `json:"name"`
	ConnectionName       string            `json:"connectionName"`
	CspImageId           string            `json:"cspImageId"`
	CspImageName         string            `json:"cspImageName"`
	Description          string            `json:"description,omitempty"`
	CreationDate         string            `json:"creationDate,omitempty"`
	GuestOS              string            `json:"guestOS,omitempty"` // Windows7, Ubuntu etc.
	Status               string            `json:"status,omitempty"`  // available, unavailable
	KeyValueList         []common.KeyValue `json:"keyValueList,omitempty"`
	AssociatedObjectList []string          `json:"associatedObjectList"`
}

// ConvertSpiderImageToTumblebugImage accepts an Spider image object, converts to and returns an TB image object
func ConvertSpiderImageToTumblebugImage(spiderImage SpiderImageInfo) (TbImageInfo, error) {
	if spiderImage.IId.NameId == "" {
		err := fmt.Errorf("ConvertSpiderImageToTumblebugImage failed; spiderImage.IId.NameId == \"\" ")
		emptyTumblebugImage := TbImageInfo{}
		return emptyTumblebugImage, err
	}

	tumblebugImage := TbImageInfo{}
	tumblebugImage.Id = spiderImage.IId.SystemId
	tumblebugImage.Name = common.LookupKeyValueList(spiderImage.KeyValueList, "Name")
	tumblebugImage.CspImageId = spiderImage.IId.SystemId
	tumblebugImage.CspImageName = common.LookupKeyValueList(spiderImage.KeyValueList, "Name")
	tumblebugImage.Description = common.LookupKeyValueList(spiderImage.KeyValueList, "Description")
	tumblebugImage.CreationDate = common.LookupKeyValueList(spiderImage.KeyValueList, "CreationDate")
	tumblebugImage.GuestOS = spiderImage.GuestOS
	tumblebugImage.Status = spiderImage.Status
	tumblebugImage.KeyValueList = spiderImage.KeyValueList

	return tumblebugImage, nil
}

// RegisterImageWithId accepts image creation request, creates and returns an TB image object
func RegisterImageWithId(nsId string, u *TbImageReq) (TbImageInfo, error) {

	resourceType := common.StrImage

	nsId = common.ToLower(nsId)
	lowerizedName := common.ToLower(u.Name)
	check, err := CheckResource(nsId, resourceType, lowerizedName)

	if check {
		temp := TbImageInfo{}
		err := fmt.Errorf("The image " + lowerizedName + " already exists.")
		return temp, err
	}

	if err != nil {
		temp := TbImageInfo{}
		err := fmt.Errorf("Failed to check the existence of the image " + lowerizedName + ".")
		return temp, err
	}

	res, err := LookupImage(u.ConnectionName, u.CspImageId)
	if err != nil {
		common.CBLog.Error(err)
		err := fmt.Errorf("an error occurred while lookup image via CB-Spider")
		emptyImageInfoObj := TbImageInfo{}
		return emptyImageInfoObj, err
	}

	content, err := ConvertSpiderImageToTumblebugImage(res)
	if err != nil {
		common.CBLog.Error(err)
		err := fmt.Errorf("an error occurred while converting Spider image info to Tumblebug image info.")
		emptyImageInfoObj := TbImageInfo{}
		return emptyImageInfoObj, err
	}
	content.ConnectionName = u.ConnectionName

	sql := "INSERT INTO `image`(" +
		"`namespace`, " +
		"`id`, " +
		"`name`, " +
		"`connectionName`, " +
		"`cspImageId`, " +
		"`cspImageName`, " +
		"`creationDate`, " +
		"`description`, " +
		"`guestOS`, " +
		"`status`) " +
		"VALUES ('" +
		nsId + "', '" +
		content.Id + "', '" +
		content.Name + "', '" +
		content.ConnectionName + "', '" +
		content.CspImageId + "', '" +
		content.CspImageName + "', '" +
		content.CreationDate + "', '" +
		content.Description + "', '" +
		content.GuestOS + "', '" +
		content.Status + "');"

	fmt.Println("sql: " + sql)
	// https://stackoverflow.com/questions/42486032/golang-sql-query-syntax-validator
	_, err = sqlparser.Parse(sql)
	if err != nil {
		return content, err
	}

	// cb-store
	fmt.Println("=========================== PUT registerImage")
	Key := common.GenResourceKey(nsId, resourceType, content.Id)
	Val, _ := json.Marshal(content)
	err = common.CBStore.Put(string(Key), string(Val))
	if err != nil {
		common.CBLog.Error(err)
		return content, err
	}
	keyValue, _ := common.CBStore.Get(string(Key))
	fmt.Println("<" + keyValue.Key + "> \n" + keyValue.Value)
	fmt.Println("===========================")

	stmt, err := common.MYDB.Prepare(sql)
	if err != nil {
		fmt.Println(err.Error())
	}
	_, err = stmt.Exec()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Data inserted successfully..")
	}

	return content, nil
}

// RegisterImageWithInfo accepts image creation request, creates and returns an TB image object
func RegisterImageWithInfo(nsId string, content *TbImageInfo) (TbImageInfo, error) {

	resourceType := common.StrImage

	nsId = common.ToLower(nsId)
	lowerizedName := common.ToLower(content.Name)
	check, err := CheckResource(nsId, resourceType, lowerizedName)

	if check {
		temp := TbImageInfo{}
		err := fmt.Errorf("The image " + lowerizedName + " already exists.")
		return temp, err
	}

	if err != nil {
		temp := TbImageInfo{}
		err := fmt.Errorf("Failed to check the existence of the image " + lowerizedName + ".")
		return temp, err
	}

	//content.Id = common.GenUuid()
	content.Id = lowerizedName
	content.Name = lowerizedName

	sql := "INSERT INTO `image`(" +
		"`namespace`, " +
		"`id`, " +
		"`name`, " +
		"`connectionName`, " +
		"`cspImageId`, " +
		"`cspImageName`, " +
		"`creationDate`, " +
		"`description`, " +
		"`guestOS`, " +
		"`status`) " +
		"VALUES ('" +
		nsId + "', '" +
		content.Id + "', '" +
		content.Name + "', '" +
		content.ConnectionName + "', '" +
		content.CspImageId + "', '" +
		content.CspImageName + "', '" +
		content.CreationDate + "', '" +
		content.Description + "', '" +
		content.GuestOS + "', '" +
		content.Status + "');"

	fmt.Println("sql: " + sql)
	// https://stackoverflow.com/questions/42486032/golang-sql-query-syntax-validator
	_, err = sqlparser.Parse(sql)
	if err != nil {
		return *content, err
	}

	fmt.Println("=========================== PUT registerImage")
	Key := common.GenResourceKey(nsId, resourceType, content.Id)
	Val, _ := json.Marshal(content)
	err = common.CBStore.Put(string(Key), string(Val))
	if err != nil {
		common.CBLog.Error(err)
		return *content, err
	}
	keyValue, _ := common.CBStore.Get(string(Key))
	fmt.Println("<" + keyValue.Key + "> \n" + keyValue.Value)
	fmt.Println("===========================")

	stmt, err := common.MYDB.Prepare(sql)
	if err != nil {
		fmt.Println(err.Error())
	}
	_, err = stmt.Exec()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Data inserted successfully..")
	}

	return *content, nil
}

type SpiderImageList struct {
	Image []SpiderImageInfo `json:"image"`
}

// LookupImageList accepts Spider conn config,
// lookups and returns the list of all images in the region of conn config
// in the form of the list of Spider image objects
func LookupImageList(connConfig string) (SpiderImageList, error) {

	if os.Getenv("SPIDER_CALL_METHOD") == "REST" {

		url := common.SPIDER_REST_URL + "/vmimage"

		// Create Req body
		type JsonTemplate struct {
			ConnectionName string `json:"ConnectionName"`
		}
		tempReq := JsonTemplate{}
		tempReq.ConnectionName = connConfig

		client := resty.New()
		client.SetAllowGetMethodPayload(true)

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(tempReq).
			SetResult(&SpiderImageList{}). // or SetResult(AuthSuccess{}).
			//SetError(&AuthError{}).       // or SetError(AuthError{}).
			Get(url)

		if err != nil {
			common.CBLog.Error(err)
			content := SpiderImageList{}
			err := fmt.Errorf("an error occurred while requesting to CB-Spider")
			return content, err
		}

		fmt.Println(string(resp.Body()))

		fmt.Println("HTTP Status code " + strconv.Itoa(resp.StatusCode()))
		switch {
		case resp.StatusCode() >= 400 || resp.StatusCode() < 200:
			err := fmt.Errorf(string(resp.Body()))
			common.CBLog.Error(err)
			content := SpiderImageList{}
			return content, err
		}

		temp := resp.Result().(*SpiderImageList)
		return *temp, nil

	} else {

		// CCM API 설정
		ccm := api.NewCloudResourceHandler()
		err := ccm.SetConfigPath(os.Getenv("CBTUMBLEBUG_ROOT") + "/conf/grpc_conf.yaml")
		if err != nil {
			common.CBLog.Error("ccm failed to set config : ", err)
			return SpiderImageList{}, err
		}
		err = ccm.Open()
		if err != nil {
			common.CBLog.Error("ccm api open failed : ", err)
			return SpiderImageList{}, err
		}
		defer ccm.Close()

		result, err := ccm.ListImageByParam(connConfig)
		if err != nil {
			common.CBLog.Error(err)
			return SpiderImageList{}, err
		}

		temp := SpiderImageList{}
		err2 := json.Unmarshal([]byte(result), &temp)
		if err2 != nil {
			fmt.Println("whoops:", err2)
		}
		return temp, nil

	}
}

// LookupImage accepts Spider conn config and CSP image ID, lookups and returns the Spider image object
func LookupImage(connConfig string, imageId string) (SpiderImageInfo, error) {

	if os.Getenv("SPIDER_CALL_METHOD") == "REST" {

		url := common.SPIDER_REST_URL + "/vmimage/" + imageId

		// Create Req body
		type JsonTemplate struct {
			ConnectionName string `json:"ConnectionName"`
		}
		tempReq := JsonTemplate{}
		tempReq.ConnectionName = connConfig

		client := resty.New()
		client.SetAllowGetMethodPayload(true)

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(tempReq).
			SetResult(&SpiderImageInfo{}). // or SetResult(AuthSuccess{}).
			//SetError(&AuthError{}).       // or SetError(AuthError{}).
			Get(url)

		if err != nil {
			common.CBLog.Error(err)
			content := SpiderImageInfo{}
			err := fmt.Errorf("an error occurred while requesting to CB-Spider")
			return content, err
		}

		fmt.Println(string(resp.Body()))

		fmt.Println("HTTP Status code " + strconv.Itoa(resp.StatusCode()))
		switch {
		case resp.StatusCode() >= 400 || resp.StatusCode() < 200:
			err := fmt.Errorf(string(resp.Body()))
			common.CBLog.Error(err)
			content := SpiderImageInfo{}
			return content, err
		}

		temp := resp.Result().(*SpiderImageInfo)
		return *temp, nil

	} else {

		// CCM API 설정
		ccm := api.NewCloudResourceHandler()
		err := ccm.SetConfigPath(os.Getenv("CBTUMBLEBUG_ROOT") + "/conf/grpc_conf.yaml")
		if err != nil {
			common.CBLog.Error("ccm failed to set config : ", err)
			return SpiderImageInfo{}, err
		}
		err = ccm.Open()
		if err != nil {
			common.CBLog.Error("ccm api open failed : ", err)
			return SpiderImageInfo{}, err
		}
		defer ccm.Close()

		result, err := ccm.GetImageByParam(connConfig, imageId)
		if err != nil {
			common.CBLog.Error(err)
			return SpiderImageInfo{}, err
		}

		temp := SpiderImageInfo{}
		err2 := json.Unmarshal([]byte(result), &temp)
		if err2 != nil {
			//fmt.Errorf("an error occurred while unmarshaling: " + err2.Error())
			common.CBLog.Error(err2)
		}
		return temp, nil

	}
}

// FetchImages gets all conn configs from Spider, lookups all images for each region of conn config, and saves into TB image objects
func FetchImages(nsId string) (connConfigCount uint, imageCount uint, err error) {
	connConfigs, err := common.GetConnConfigList()
	if err != nil {
		common.CBLog.Error(err)
		return 0, 0, err
	}

	for _, connConfig := range connConfigs.Connectionconfig {
		fmt.Println("connConfig " + connConfig.ConfigName)

		spiderImageList, err := LookupImageList(connConfig.ConfigName)
		if err != nil {
			common.CBLog.Error(err)
			return 0, 0, err
		}

		for _, spiderImage := range spiderImageList.Image {
			tumblebugImage, err := ConvertSpiderImageToTumblebugImage(spiderImage)
			if err != nil {
				common.CBLog.Error(err)
				return 0, 0, err
			}

			tumblebugImageId := connConfig.ConfigName + "-" + tumblebugImage.Name
			//fmt.Println("tumblebugImageId: " + tumblebugImageId) // for debug

			check, err := CheckResource(nsId, common.StrImage, tumblebugImageId)
			if check {
				common.CBLog.Infoln("The image " + tumblebugImageId + " already exists in TB; continue")
				continue
			} else if err != nil {
				common.CBLog.Infoln("Cannot check the existence of " + tumblebugImageId + " in TB; continue")
				continue
			} else {
				tumblebugImage.Id = tumblebugImageId
				tumblebugImage.Name = tumblebugImageId
				tumblebugImage.ConnectionName = connConfig.ConfigName

				_, err := RegisterImageWithInfo(nsId, &tumblebugImage)
				if err != nil {
					common.CBLog.Error(err)
					return 0, 0, err
				}
			}
			imageCount++
		}
		connConfigCount++
	}
	return connConfigCount, imageCount, nil
}

// SearchImage accepts arbitrary number of keywords, and returns the list of matched TB image objects
func SearchImage(nsId string, keywords ...string) ([]TbImageInfo, error) {
	nsId = common.GenId(nsId)

	tempList := []TbImageInfo{}

	sqlQuery := "SELECT * FROM `image` WHERE `namespace`='" + nsId + "'"

	for _, keyword := range keywords {
		//fmt.Println("in SearchImage(); keyword: " + keyword) // for debug
		keyword = common.GenId(keyword)
		sqlQuery += " AND `name` LIKE '%" + keyword + "%'"
	}
	sqlQuery += ";"
	_, err := sqlparser.Parse(sqlQuery)
	if err != nil {
		return tempList, err
	}

	rows, err := common.MYDB.Query(sqlQuery)
	if err != nil {
		common.CBLog.Error(err)
		return tempList, err
	}

	cols, err := rows.Columns()
	if err != nil {
		common.CBLog.Error(err)
		return tempList, err
	}

	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return tempList, err
		}
		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			m[colName] = *val
		}
		js, _ := json.Marshal(m)
		tempImage := TbImageInfo{}
		err = json.Unmarshal(js, &tempImage)
		if err != nil {
			common.CBLog.Error(err)
			return nil, err
		}
		tempList = append(tempList, tempImage)
	}
	return tempList, nil
}
