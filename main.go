package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
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
service {{.ServiceName}}Svc {
	{{.AppendService}}
}
message {{.ServiceArgName}} {
	{{range $i ,$v :=.Parameters -}}
	{{$v.Type }}  {{ $v.Name }} = {{ AddOne $i }};
	{{end}}
}
{{range $msgName,$item := .ResponseData -}}
message {{$msgName}} {
	{{$inx := Var 0 -}}
	{{range $i,$v := $item -}}
		{{$inx.Set (AddOne $inx.Value) -}}
		{{$v}} {{$i}} = {{$inx.Value -}};
	{{end}}
}
{{end}}
`

type TemplateValue struct {
	PackageName    string
	ServiceName    string
	ServiceArgName string
	AppendService  string
	Parameters     []Parameter
	ResponseData   map[string]interface{}
}

var objectId int
var subObject map[string]interface{}

func main() {
	objectId = 1
	subObject = make(map[string]interface{})
	isAppend := true
	interfaceName := "/hdmp/common/block"
	packageName := "cleaned"
	serviceName := "CommunityCleaned"
	method := "Find"
	serviceArgName := "RequestById"
	serviceReturnName := serviceName
	appendService := fmt.Sprintf("rpc %s (%s) returns (%s) {}", method, serviceArgName, serviceReturnName)
	data, err := ioutil.ReadFile("swagger.json")
	if err != nil {
		log.Fatal(err)
	}
	var swagger Swagger
	json.Unmarshal(data, &swagger)
	// fmt.Println(swagger.Paths)
	// paramters := swagger.Paths["/hdmp/common/block"].Get.Parameters
	paramters := swagger.Paths[interfaceName].Get.Parameters
	fmt.Println(paramters)
	//TODO other data  type
	for pk, pv := range paramters {
		if pv.Type == "integer" {
			paramters[pk].Type = "string"
		}
	}
	responses := swagger.Paths[interfaceName].Get.Response["200"]
	// fmt.Printf("%v", responses.Schema["$ref"])
	definitionIndex := strings.Split(responses.Schema["$ref"], "/")
	index := definitionIndex[len(definitionIndex)-1]
	responseProperties := swagger.Definitions[index].Properties
	//TODO 根据类型判断
	responseData := responseProperties["data"].Example
	responseRes := make(map[string]interface{})
	handleResponse(responseData, &responseRes, serviceReturnName)

	fmt.Println(responseRes)
	templateValue := TemplateValue{packageName, serviceName, serviceArgName, appendService, paramters, responseRes}
	tmpl := template.New("proto")
	tmpl.Funcs(template.FuncMap{"AddOne": addOne, "Var": newVariable})
	tmpl.Parse(tpl)
	filename := "./tmpProto"
	var f *os.File
	var err1 error
	var line string
	if checkFileIsExist(filename) { //如果文件存在
		f, err1 = os.OpenFile(filename, os.O_RDWR, os.ModePerm)
		// f, err1 = os.Open(filename)
		rd := bufio.NewReader(f)
		var err2 error
		var offIndex int
		var rest []byte
		for {
			line, err2 = rd.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				break
			}
			offIndex += len(line)
			if strings.HasPrefix(strings.Trim(line, ""), "service") {
				rest, _ = ioutil.ReadAll(rd)
				var buffer bytes.Buffer
				buffer.Write([]byte("serviceddfs\n"))
				buffer.Write(rest)
				rest = buffer.Bytes()
				break
			}
			// fmt.Println(string(line))
		}
		if isAppend {
			f.WriteAt(rest, int64(offIndex)+1)
		}
	} else {
		f, err1 = os.Create(filename) //创建文件
		tmpl.Execute(f, templateValue)
	}

	// err1 = tmpl.Execute(os.Stdout, templateValue)

	if err1 != nil {
		log.Fatal(err1)
	}
}
func addOne(i int) int {
	return i + 1
}

type variable struct {
	Value interface{}
}

func (v *variable) Set(value interface{}) string {
	v.Value = value
	return ""
}

func newVariable(initialValue interface{}) *variable {
	return &variable{initialValue}
}

func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}
func getObjectID(objectID *int) int {
	tmp := *objectID
	*objectID++
	return tmp
}
func handleResponse(data map[string]interface{}, res *map[string]interface{}, objName string) {
	var valueType interface{}
	tempRes := make(map[string]interface{})
	for key, value := range data {
		valueType = reflect.ValueOf(value).Kind()
		if valueType == reflect.Slice {
			responseArr := value.([]interface{})
			tempData := responseArr[0].(map[string]interface{})
			// fmt.Println(tempData)
			subKey := "ResponseObject" + strconv.Itoa(getObjectID(&objectId))
			subObject[subKey] = responseArr[0]
			handleResponse(tempData, res, subKey)
			tempRes[key] = "repeated " + subKey
		} else if valueType == reflect.Map {
			tempRes[key] = "ResponseObject"
		} else {
			tempRes[key] = "string"
		}
		(*res)[objName] = tempRes
	}
}
