package main

import (
	"bytes"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/emicklei/proto"
)

const tmpl = `type: google.api.Service
config_version: 3

http:
 rules:{{ range $item := . }}
 - selector: {{$item.PkgName}}.{{$item.ServiceName}}.{{$item.MethodName}}
   post: /{{$item.Prefix}}{{$item.PkgName}}/{{$item.ServiceName}}/{{$item.MethodName}}
   body: "*"{{ end }}
`

var (
	input, output, prefix string
)

func init() {
	flag.StringVar(&input, "i", "", "input proto file dir")
	flag.StringVar(&output, "o", "", "output config yaml")
	flag.StringVar(&output, "p", "", "prefix for url")
	flag.Parse()

	if input == "" || output == "" {
		log.Fatal(flag.ErrHelp)
	}

}

func main() {
	files, err := filepath.Glob(input)
	if err != nil {
		log.Fatal(err)
	}

	if len(files) == 0 {
		log.Fatal("not found any file *.proto")
	}

	var allMethods []rpcService
	for _, f := range files {
		log.Println("parse file: ", f)
		methods, err := readProtoFile(f)
		if err == nil {
			allMethods = append(allMethods, methods...)
		}
	}

	t := template.Must(template.New("tmpl").Parse(tmpl))

	var outFile bytes.Buffer
	err = t.Execute(&outFile, allMethods)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(output, outFile.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}

type rpcService struct {
	Prefix, PkgName, ServiceName, MethodName, Request, Response string
}

func readProtoFile(dir string) ([]rpcService, error) {
	reader, _ := os.Open(dir)
	defer reader.Close()

	parser := proto.NewParser(reader)
	definition, _ := parser.Parse()

	var (
		methods              []rpcService
		pkgName, serviceName string
	)

	proto.Walk(definition,
		func(v proto.Visitee) {
			if s, ok := v.(*proto.Package); ok {
				pkgName = s.Name
			}
		},
		proto.WithService(func(service *proto.Service) {
			serviceName = service.Name
		}),
		proto.WithRPC(func(rpc *proto.RPC) {
			methods = append(methods, rpcService{
				Prefix:      prefix,
				PkgName:     pkgName,
				ServiceName: serviceName,
				MethodName:  rpc.Name,
				Request:     rpc.RequestType,
				Response:    rpc.ReturnsType,
			})
		}),
	)

	return methods, nil
}
