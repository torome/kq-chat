package datav

import (
	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"
	"cjgt/app/bi/cmd/rpc/bi"
	"cjgt/common/xerr"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"log"
	"strings"
	"time"
)

type TransformLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type FieldMapping struct {
	TargetField  string `json:"targetField"`
	SourceField  string `json:"sourceField"`
	DefaultValue string `json:"defaultValue"`
	Type         string `json:"type"`
}

type APIConfig struct {
	Type    string              `json:"type"` // api, mock, db, excel...
	URL     string              `json:"url"`
	IsLogin int64               `json:"isLogin"`
	Method  string              `json:"method"`
	Params  []map[string]string `json:"params"` // 会预留一些对象id， obj_id, obj2_id...正常一个够用了 objId
	Body    map[string]string   `json:"body"`
	Mapping []FieldMapping      `json:"mapping"`
}

type GroupConfig struct {
	URL    string         `json:"url"`
	Code   int64          `json:"code"`
	Msg    string         `json:"msg"`
	Object string         `json:"object"` // 会预留一些对象id， obj_id, obj2_id...正常一个够用了 objId
	List   string         `json:"list"`
	Params []FieldMapping `json:"params"`
	Body   []FieldMapping `json:"body"`
	Header []FieldMapping `json:"header"`
}

type Mapping struct {
	TargetField     string `json:"targetField"`
	SourceField     string `json:"sourceField"`
	IsArrayHandling string `json:"isArrayHandling"`
	DefaultValue    string `json:"defaultValue"`
}

type Obj interface{} // 空接口，可以接收任何类型的值

type Response struct {
	Obj Obj `json:"obj"`
}

func NewTransformLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TransformLogic {
	return &TransformLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TransformLogic) Transform(req *types.TransformReq) (*types.TransformResp, error) {
	// 下发数据包含两部分，一部分是配置，一部分是数据
	// 配置主要是定义请求参数初始化的值，以及其他必要的配置信息（根据前端图标需要）
	// 数据主要是格式化几种类型的数据规范

	var uuid string

	params := req.Params
	//msg.Id = body["uuid"].(string)
	uuid = req.Uuid

	resp, err := l.svcCtx.BiRpc.DatasetDetail(l.ctx, &bi.DatasetDetailReq{
		Uuid: uuid,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("get homestay order detail fail"), " rpc get HomestayOrderDetail err:%v , sn : %s", err, req.Uuid)
	}

	//jsonConfig := `{"name": "John", "age": 30}`
	//jsonData := `{"name": "John", "age": 30}`

	var conf APIConfig
	fmt.Println("111")
	fmt.Println(resp.Dataset.Config)
	if err = json.Unmarshal([]byte(resp.Dataset.Config), &conf); err != nil {
		log.Fatal(err)
	}
	fmt.Println("222")
	jsonMeta := make(map[string]interface{})
	jsonExtra := make(map[string]interface{})

	jsonExtra["total"] = 26

	if err := json.Unmarshal([]byte(resp.Dataset.Meta), &jsonMeta); err != nil {
		fmt.Println("---->")
		log.Fatal(err)
	}

	var jsonData interface{}

	// test start

	// test end

	if conf.Type == "api" {
		// 调用转换函数进行数据转换
		fmt.Printf("resp: %v", params)
		jsonData, err = transformData(conf, params)
		if err != nil {
			//log.Fatal(err)
			return nil, err
		}
		fmt.Printf("Transformed Array Data: %s\n", jsonData)
	}

	if conf.Type == "mock" {
		if err := json.Unmarshal([]byte(resp.Dataset.Data), &jsonData); err != nil {
			log.Fatal(err)
		}
	}

	return &types.TransformResp{
		Uuid:   uuid,
		Remark: resp.Dataset.Remark,
		Meta:   jsonMeta,
		Data:   jsonData,
		Extra:  jsonExtra,
	}, nil
}

func transformData(apiConfig APIConfig, params map[string]string) (interface{}, error) {

	// groupConf 配置的信息，针对api类型，包括接口地址，请求参数名列表，返回数据的通用部分（具体拆分就是返回状态，返回消息，返回数据3部分）
	// 针对API情况，核心就是请求过程的通用部分进行配置
	/*
		{
			"url": "http://47.103.78.79:8091",
			"code": "api",
			"msg": "your_token_value",
			"object": "",
			"list": "",
			"params": [{
				"type": "text",
				"format": "",
				"insideField": "token",
				"defaultValue": "",
				"externalField": "tokenId"
			}]
		}
	*/
	var transformedData interface{}

	// 内部走统一的参数约定，所以这里需要进行转换参数名和数据格式，暂时针对日期的几种情况
	// 年2024，年月2024-04，年月日2024-04-24
	// 结果映射
	externalParams := make(map[string]string)

	fmt.Printf("params: %v", params)
	fmt.Printf("apiConfig.Params: %v", apiConfig.Params)
	// 遍历每个映射配置项
	for _, config := range apiConfig.Params {
		switch config["type"] {
		case "date":
			internalDate := params[config["insideField"]]
			externalDate := convertDate(internalDate, config["format"])
			externalParams[config["externalField"]] = externalDate

		default:
			externalParams[config["externalField"]] = params[config["insideField"]]
		}
	}

	// 输出转换后的外部参数
	fmt.Printf("externalParams: %v", externalParams)

	client := resty.New() // 创建一个restry客户端
	//resp, err := client.R().EnableTrace().Get("https://httpbin.org/get")
	// 发送 HTTP 请求并获取数据，这里使用示例库 go-resty 进行请求
	resp, err := client.R().
		SetQueryParams(externalParams).
		SetBody(apiConfig.Body).
		SetHeader("Accept", "application/json").
		//SetHeader("tokenId", "e763216f-e2cd-4eac-990e-e245fda3ec7a"). // 添加 token 到请求头
		Execute(apiConfig.Method, apiConfig.URL)
	if err != nil {
		return transformedData, err
	}

	// fmt.Printf("resp: %v", resp)

	type Result struct {
		Success bool          `json:"success"`
		Msg     string        `json:"msg"`
		Obj     interface{}   `json:"obj"`  // 单个对象和多个对象的情况
		Rows    []interface{} `json:"rows"` // 带分页的情况
	}

	var ResultData Result

	err = json.Unmarshal(resp.Body(), &ResultData)
	if err != nil {
		return transformedData, err
	}

	if ResultData.Success == false {
		return nil, errors.Wrapf(xerr.NewErrMsg(ResultData.Msg), "create homestay order rpc CreateHomestayOrder fail req: %+v , err : %v ", params, err)
	}
	// 针对登录场景，特殊处理
	if apiConfig.IsLogin == 1 {

		data2 := ResultData.Obj

		var value1 string
		// 使用类型断言将 data 断言为 map[string]interface{}
		// 这里我们使用逗号，ok 的形式来检查断言是否成功
		if mappedData, ok := data2.(map[string]interface{}); ok {
			// 如果 ok 为 true，那么 mappedData 就是我们想要的 map[string]interface{}
			// 现在我们可以读取它的内容了
			name, ok := mappedData["custName"].(string)
			if ok {
				fmt.Println("Name:", name)
			}

			value1 = name

		} else {
			// 如果 data 不是 map[string]interface{} 类型，那么这里会执行
			fmt.Println("data is not a map[string]interface{}")
		}

		var (
			a = map[string]interface{}{
				"value0": ResultData.Msg,
				"value1": value1,
			}
		)

		return a, nil
	}

	var tmp interface{}
	if ResultData.Obj != nil {
		tmp = ResultData.Obj
	}
	if ResultData.Rows != nil {
		tmp = ResultData.Rows
	}

	switch input := tmp.(type) {
	case []interface{}:
		// 处理数组类型
		transformed := make([]interface{}, len(input))
		for i, v := range input {
			item, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected array element to be an object, got: %T", v)
			}
			transformedItem, err := transformObject(item, apiConfig.Mapping)
			if err != nil {
				return nil, err
			}
			transformed[i] = transformedItem
		}
		return transformed, nil
	case map[string]interface{}:
		// 处理对象类型
		return transformObject(input, apiConfig.Mapping)
	default:
		return nil, fmt.Errorf("unsupported input type: %T", input)
	}
}

func transformObject(input map[string]interface{}, mappings []FieldMapping) (map[string]interface{}, error) {
	transformed := make(map[string]interface{})
	// 针对坐标数据的相关处理
	latitude := ""
	longitude := ""
	coordinateField := ""

	// 针对链接地址，比如图片等，统一转换成内部的代理地址输出，因为考虑外部可能是http，而我们都是走https，会导致图片显示不正常的问题 todo...

	for _, mapping := range mappings {
		if value, ok := input[mapping.SourceField]; ok {
			if arr, isArray := value.([]interface{}); isArray && len(arr) > 0 {
				// 如果是数组类型且非空，取第一个元素进行转换
				url := ToString(arr[0])
				// 临时替换，待改进
				if strings.HasPrefix(url, "http://47.103.78.79:8091/") {
					// 替换http://为https://
					url = strings.Replace(url, "http://47.103.78.79:8091/", "https://oss.17886.cn/", 1)
				}
				transformed[mapping.TargetField] = url
			} else if v, isNum := value.(json.Number); isNum {
				// 如果是数值类型，将其转换为字符串
				transformed[mapping.TargetField] = v.String()
			} else if mapping.Type == "longitude" {
				fmt.Printf("mapping333 : %v", value)
				// 返回null情况
				if value != nil {
					longitude = ToString(value)
				}
				coordinateField = mapping.TargetField

			} else if mapping.Type == "latitude" {
				if value != nil {
					latitude = ToString(value)
				}
				coordinateField = mapping.TargetField

			} else {
				transformed[mapping.TargetField] = value
			}
		} else {
			transformed[mapping.TargetField] = mapping.DefaultValue
		}

		if coordinateField != "" {
			if latitude != "" && longitude != "" {
				transformed[coordinateField] = longitude + "," + latitude
			} else {
				transformed[coordinateField] = ""
			}
		}

	}
	return transformed, nil
}

func ToString(v interface{}) string {
	return fmt.Sprint(v)
}

func convertDate(internalDate, format string) string {
	parsedTime, err := parseTimeAttempts(internalDate)
	if err != nil {
		return ""
	}

	switch format {
	case "yyyy":
		return parsedTime.Format("2006")
	case "mm":
		return parsedTime.Format("01")
	case "dd":
		return parsedTime.Format("02")
	case "yyyy-mm":
		return parsedTime.Format("2006-01")
	case "yyyy-mm-dd":
		return parsedTime.Format("2006-01-02")
	default:
		return ""
	}
}

// parseTimeAttempts 尝试使用多种格式来解析时间字符串
func parseTimeAttempts(timeStr string) (time.Time, error) {
	// 尝试的多种时间布局
	layouts := []string{
		"2006",       // 只包含年份
		"2006-01",    // 年份和月份
		"2006-01-02", // 年份、月份和日期
		//"2006-01-02 15:04:05", // 完整的时间格式，这里可能不需要，因为输入不包含时间
	}

	// 遍历每种布局并尝试解析
	for _, layout := range layouts {
		t, err := time.Parse(layout, timeStr)
		if err == nil {
			// 如果解析成功，返回时间对象
			return t, nil
		}
	}

	// 如果所有布局都失败，返回错误
	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}
