package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Swagger struct {
	Paths       map[string]Service   `json:"paths"`
	Definitions map[string]DeContent `json:"definitions"`
}
type Service struct {
	Get  Method `json:"get"`
	Post Method `json:"post"`
	Put  Method `json:"put"`
}
type Method struct {
	Tags       []string            `json:"tags"`
	Summary    string              `json:"summary"`
	Parameters []Parameter         `json:"parameters"`
	Response   map[string]Response `json:"responses"`
}
type Parameter struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Schema   Schema `json:"schema"`
	Type     string `json:"type"`
}
type Response struct {
	Schema      map[string]string `json:"schema"`
	Description string            `json:"description"`
}
type ResponseItems struct {
	Description string `json:"description"`
}
type Schema struct {
	Required   []string            `json:"required"`
	Properties map[string]Property `json:"properties"`
}
type Property struct {
	Type    string                 `json:"type"`
	Default interface{}            `json:"default"`
	Example map[string]interface{} `json:"example"`
}
type DeContent struct {
	Title      string              `json:"title"`
	Properties map[string]Property `json:"properties"`
}

const tpl = `
syntax = "proto3";
package {{.PackageName}};
service CommunityCleanedSvc {
    rpc Find (RequestById) returns (CommunityCleaned) {}
    rpc Search (RequestByQuery) returns (CommunityCleanedList) {}
}
message RequestById {
    string id = 1;
    string filter = 2;
}
message RequestByQuery {
	{{range $i ,$v :=.Parameters}}
	{{$v.Type}}  {{$v.Name}} = {{AddOne $i}}
	{{end}}
}
message CommunityCleaned {
	{{range $i,$v :=.ResponseData}}
		{{$v}} {{$i}}
	{{end}}
}
message CommunityCleanedList {
    string total_record_num = 1;
    string total_page_num = 2;
    repeated CommunityCleaned data = 3;
}
`

type TemplateValue struct {
	PackageName  string
	Parameters   []Parameter
	ResponseData map[string]interface{}
}

var objectId int

func main() {
	objectId = 1
	data, err := ioutil.ReadFile("swagger.json")
	if err != nil {
		log.Fatal(err)
	}
	var swagger Swagger
	json.Unmarshal(data, &swagger)
	// fmt.Println(swagger.Paths)
	paramters := swagger.Paths["/hdmp/common/block"].Get.Parameters
	// fmt.Println(swagger.Paths)
	responses := swagger.Paths["/hdmp/common/block"].Get.Response["200"]
	// fmt.Printf("%v", responses.Schema["$ref"])
	definitionIndex := strings.Split(responses.Schema["$ref"], "/")
	index := definitionIndex[len(definitionIndex)-1]
	responseProperties := swagger.Definitions[index].Properties
	//TODO 根据类型判断
	fmt.Println(responseProperties["data"].Type)
	responseData := responseProperties["data"].Example

	responseRes := handleResponse(responseData)

	fmt.Printf("objectId:%d \n", objectId)
	fmt.Println(responseRes)
	templateValue := TemplateValue{"hello", paramters, responseData}
	tmpl := template.New("proto")
	tmpl.Funcs(template.FuncMap{"AddOne": addOne})
	tmpl.Parse(tpl)
	filename := "./tmpProto"
	var f *os.File
	var err1 error
	if checkFileIsExist(filename) { //如果文件存在
		f, err1 = os.OpenFile(filename, os.O_APPEND, 0666) //打开文件
	} else {
		f, err1 = os.Create(filename) //创建文件
	}
	if err1 != nil {
		log.Fatal(err1)
	}
	tmpl.Execute(f, templateValue)
	// err1 = tmpl.Execute(os.Stdout, templateValue)
	if err1 != nil {
		log.Fatal(err1)
	}
	// for index, paramter := range paramters {
	// 	fmt.Println(index)
	// 	fmt.Println(paramter.Name)

	// }

	// for i,u = range  s
	// m := docs.(map[string]interface{})
	// for i, u := range m["paths"].(map[string]interface{}) {
	// 	fmt.Println(i)
	// 	fmt.Println(u)
	// }

	// for k, v := range m {
	// 	switch vv := v.(type) {
	// 	case string:
	// 		fmt.Println(k, "is string", vv)
	// 	case int:
	// 		fmt.Println(k, "is int", vv)
	// 	case []interface{}:
	// 		fmt.Println(k, "is an array:")
	// 		for i, u := range vv {
	// 			fmt.Println(i, u)
	// 		}
	// 	default:
	// 		fmt.Println(k, "is of a type I don't know how to handle")
	// 	}
	// }
}
func addOne(i int) int {
	return i + 1
}

func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}
func getObjectId(objectId *int) int {
	tmp := *objectId
	*objectId++
	return tmp
}
func handleResponse(responseData map[string]interface{}) map[string]interface{} {
	responseRes := make(map[string]interface{})
	subObject := make(map[string]interface{})
	var valueType interface{}
	//now can't support n (n>2) demensions
	for key, value := range responseData {
		valueType = reflect.ValueOf(value).Kind()
		if valueType == reflect.Slice {
			responseArr := value.([]interface{})
			tempData := responseArr[0].(map[string]interface{})
			for subKey, subVal := range tempData {
				fmt.Println(subVal)
				subObject[subKey] = "string"
			}
			responseRes[key] = "repeated ResponseObject" + strconv.Itoa(getObjectId(&objectId))
		} else if valueType == reflect.Map {
			responseRes[key] = "ResponseObject"
		} else {
			responseRes[key] = "string"
		}
	}
	return responseRes
}
