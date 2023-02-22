package util

import (
	"encoding/json"
	"fmt"
	"github.com/patrickmn/go-cache"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"time"
)

var (
	tokenCache = cache.New(time.Minute*20, time.Minute*20)
)

func Cmd(cmd string) (string, int) {
	fmt.Println(cmd)
	cmdObj := exec.Command("bash", "-c", cmd)
	output, err := cmdObj.CombinedOutput()
	if err != nil {
		return err.Error(), 1
	}
	return string(output), 0
}

func DockerCmd(id, cmd string) (string, int) {
	return Cmd(fmt.Sprintf("docker exec %s %s", id, cmd))
}

func HttpHealthCheck(port int, endpoint string) bool {
	c := http.DefaultClient
	c.Timeout = time.Millisecond * 500
	resp, err := c.Get(fmt.Sprintf("http://localhost:%d%s", port, endpoint))
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func NacosHealthCheck(service, namespace, username, password string) bool {

	token, ok := tokenCache.Get(namespace)
	if !ok {
		token = GeneratorNacosToken(username, password)
		tokenCache.Set(namespace, token, cache.DefaultExpiration)
	}

	u := fmt.Sprintf("http://localhost:8848/nacos/v1/ns/catalog/instances?&accessToken=%s"+
		"&serviceName=%s"+
		"&clusterName=DEFAULT"+
		"&groupName=DEFAULT_GROUP"+
		"&pageSize=10&pageNo=1"+
		"&namespaceId=%s", token, service, namespace)

	resp, err := http.Get(u)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		return false
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var rs map[string]interface{}
	err = json.Unmarshal(bytes, &rs)
	if err != nil {
		log.Fatal(err)
	}
	if int(rs["count"].(float64)) == 1 {
		return true
	}
	return false
}

func GeneratorNacosToken(username, password string) string {

	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	resp, err := http.PostForm("http://localhost:8848/nacos/v1/auth/users/login", form)
	if err != nil {
		log.Fatal(err)
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var rs map[string]interface{}
	err = json.Unmarshal(bytes, &rs)
	if err != nil {
		log.Fatal(err)
	}
	return rs["accessToken"].(string)

}
