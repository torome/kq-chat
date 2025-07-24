package datav

import (
	"cjgt/app/bi/cmd/rpc/bi"
	"cjgt/common/xerr"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"log"

	"cjgt/app/bi/cmd/api/internal/svc"
	"cjgt/app/bi/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (*types.LoginResp, error) {
	// 暂时先转发，后续要读取对应项目的配置信息，判断是否需要登录，登录的规则是什么

	// 下发数据包含两部分，一部分是配置，一部分是数据
	// 配置主要是定义请求参数初始化的值，以及其他必要的配置信息（根据前端图标需要）
	// 数据主要是格式化几种类型的数据规范

	var uuid string

	body := req.Params
	fmt.Println("-------------->")
	fmt.Println(body)
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

	if err = json.Unmarshal([]byte(resp.Dataset.Config), &conf); err != nil {
		log.Fatal(err)
	}

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

		jsonData, err = l.transformData(conf, body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Transformed Array Data: %s\n", jsonData)
	}

	if conf.Type == "mock" {
		if err := json.Unmarshal([]byte(resp.Dataset.Data), &jsonData); err != nil {
			log.Fatal(err)
		}
	}

	return &types.LoginResp{
		Uuid:   uuid,
		Remark: resp.Dataset.Remark,
		Meta:   jsonMeta,
		Data:   jsonData,
		Extra:  jsonExtra,
	}, nil
}

func (l *LoginLogic) transformData(apiConfig APIConfig, params map[string]string) (interface{}, error) {

	var transformedData interface{}

	client := resty.New() // 创建一个restry客户端
	//resp, err := client.R().EnableTrace().Get("https://httpbin.org/get")
	// 发送 HTTP 请求并获取数据，这里使用示例库 go-resty 进行请求
	fmt.Printf("apiConfig.Body: %v", apiConfig.Body)
	resp, err := client.R().
		SetQueryParams(params).
		SetBody(apiConfig.Body).
		SetHeader("Accept", "application/json").
		SetHeader("tokenId", "e763216f-e2cd-4eac-990e-e245fda3ec7a"). // 添加 token 到请求头
		Execute(apiConfig.Method, apiConfig.URL)
	if err != nil {
		fmt.Println("======", err)
		return transformedData, err
	}
	fmt.Printf("resp: %v", resp)

	// 需要考虑接收完整的resp，因为不同的接口，resp格式不同，需要走配置进行映射，而不是固定结构体
	type Result struct {
		Msg  string        `json:"msg"`
		Obj  interface{}   `json:"obj"`  // 单个对象和多个对象的情况
		Rows []interface{} `json:"rows"` // 带分页的情况
	}

	var ResultData Result

	err = json.Unmarshal(resp.Body(), &ResultData)
	if err != nil {
		return transformedData, err
	}

	if apiConfig.URL == "http://49.7.235.86:8091/YGY/cdbService/api/login" {
		var (
			a = map[string]interface{}{
				"value0": ResultData.Msg,
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
			transformedItem, err := l.transformObject(item, apiConfig.Mapping)
			if err != nil {
				return nil, err
			}
			transformed[i] = transformedItem
		}
		return transformed, nil
	case map[string]interface{}:
		// 处理对象类型
		return l.transformObject(input, apiConfig.Mapping)
	default:
		return nil, fmt.Errorf("unsupported input type: %T", input)
	}
}

func (l *LoginLogic) transformObject(input map[string]interface{}, mappings []FieldMapping) (map[string]interface{}, error) {
	transformed := make(map[string]interface{})
	for _, mapping := range mappings {
		if value, ok := input[mapping.SourceField]; ok {
			if arr, isArray := value.([]interface{}); isArray && len(arr) > 0 {
				// 如果是数组类型且非空，取第一个元素进行转换
				transformed[mapping.TargetField] = arr[0]
			} else if v, isNum := value.(json.Number); isNum {
				// 如果是数值类型，将其转换为字符串
				transformed[mapping.TargetField] = v.String()
			} else {
				transformed[mapping.TargetField] = value
			}
		} else {
			transformed[mapping.TargetField] = mapping.DefaultValue
		}
	}
	return transformed, nil
}
