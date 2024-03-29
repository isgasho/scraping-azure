package scraper

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/gocolly/colly"
	"github.com/thedevsaddam/gojsonq"
)

type resource struct {
	Name string
	Spec interface{}
}

func DownloadSpec() error {
	fileUrl := "https://docs.microsoft.com/en-us/azure/templates/toc.json"
	filePath := "toc.json"

	resp, err := http.Get(fileUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func Unwrapper() {
	json, err := ioutil.ReadFile("./toc.json")
	if err != nil {
		fmt.Println("No way!")
	}

	jsonQuery := gojsonq.New().JSONString(string(json))
	totalRes := jsonQuery.Find("items.[1].children")

	if jsonQuery.Error() != nil {
		log.Fatal(jsonQuery.Errors())
	}

	for i := 0; i < reflect.ValueOf(totalRes).Len(); i++ {
		iq := gojsonq.New().JSONString(string(json))
		mother := fmt.Sprintf("items.[1].children.[%d].children.[1].children", i)
		max := iq.Find(mother)
		for j := 0; j < reflect.ValueOf(max).Len(); j++ {
			jq := gojsonq.New().JSONString(string(json))
			path := jq.Find(mother + fmt.Sprintf(".[%d].href", j))
			new, url, err := getSpec(fmt.Sprintf("%s", path))
			if err != nil {
				fmt.Println(err)
			}
			saveSpec(new, url)
		}
	}
}

func saveSpec(spec string, url string) {
	if spec == "" || url == "" {
		fmt.Println("Can't be!")
	}

	path := "https://docs.microsoft.com/en-us/azure/templates/"
	resource := url[strings.LastIndex(url, "/")+1:]

	var dir strings.Builder
	dir.WriteString("azure_templates/")
	dir.WriteString(strings.TrimSuffix(strings.TrimPrefix(url, path), resource))

	if _, err := os.Stat(dir.String()); os.IsNotExist(err) {
		if err := os.MkdirAll(dir.String(), os.ModePerm); err != nil {
			fmt.Println("We have a problem!")
		}
	}

	var file strings.Builder
	file.WriteString(dir.String() + resource + ".json")

	if err := ioutil.WriteFile(file.String(), []byte(spec), os.ModePerm); err != nil {
		panic(err)
	}
}

func getSpec(path string) (resource string, resURL string, err error) {
	var url strings.Builder
	url.WriteString("https://docs.microsoft.com/en-us/azure/templates/" + path)

	c := colly.NewCollector()

	c.OnHTML("code.lang-json", func(e *colly.HTMLElement) {
		resource = fmt.Sprintf("%v", *e)
		resource = strings.TrimPrefix(resource, "{code ")
		resource = resource[:strings.LastIndex(resource, "[{ class lang-json}]")]
	})

	if err := c.Visit(url.String()); err != nil {
		return "", "", err
	}

	return resource, url.String(), nil
}
