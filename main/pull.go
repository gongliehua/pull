package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// 解析执行文件后面的参数
var Args = make(map[string]string)

// 解析JSON配置
var Config struct {
	Homepage string
	Port     string
	Token    string
	Repo     []interface{}
}

func init() {
	// 解析执行文件后面的参数
	for k, v := range os.Args {
		if k == 0 {
			continue
		}
		if k%2 == 0 {
			Args[os.Args[k-1]] = v
		}
	}

	// 获取JSON配置文件地址
	path, ok := Args["-c"]
	if !ok {
		log.Fatal("请使用 -c 指定配置文件")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}
	// 读取文件内容
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	// JSON配置文件内容解码
	err = json.Unmarshal(content, &Config)
	if err != nil {
		log.Fatal(err)
	}
}

// 判断当前分支
func isBranch(dir string, branch string) (bool, error) {
	cmd := exec.Command("git", "branch")
	cmd.Dir = dir

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return false, err
	}

	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		if ok, err := regexp.MatchString(`^\*\s`+branch+`$`, line); ok && err == nil {
			return true, nil
		}
	}

	return false, errors.New("未找当前分支或仓库目前不在该分支上")
}

// 拉取代码
func gitPull(dir string, branch string) (bool, error) {
	cmd := exec.Command("git", "pull", "origin", branch)
	cmd.Dir = dir

	err := cmd.Run()
	if err != nil {
		return false, err
	}

	return true, nil
}

// 首页
func indexFunc(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<a href="`+Config.Homepage+`" target="_blank">`+Config.Homepage+`</a>`)
}

// 拉取代码
func pullFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fmt.Fprintf(w, "为了安全起见，请使用 POST 请求。")
		return
	}

	token := r.FormValue("token")
	if token == "" {
		fmt.Fprintf(w, "缺少 token 参数")
		return
	}
	if token != Config.Token {
		fmt.Fprintf(w, "token 参数错误，请联系管理员")
		return
	}

	slug := r.FormValue("slug")
	if slug == "" {
		fmt.Fprintf(w, "缺少 slug 参数")
		return
	}
	for _, v := range Config.Repo {
		repo := v.(map[string]interface{})
		if repo["Slug"] == slug {
			dir := repo["Dir"].(string)
			branch := repo["Branch"].(string)

			ok, err := isBranch(dir, branch)
			if !ok || err != nil {
				fmt.Fprintf(w, "操作失败，错误消息：%s", err.Error())
				return
			}
			ok, err = gitPull(dir, branch)
			if !ok || err != nil {
				fmt.Fprintf(w, "操作失败，错误消息：%s", err.Error())
				return
			}

			fmt.Fprintf(w, "操作完成")
			return
		}
	}
	fmt.Fprintf(w, "slug 参数错误，请联系管理员")
}

func main() {
	http.HandleFunc("/", indexFunc)
	http.HandleFunc("/pull", pullFunc)
	log.Fatal(http.ListenAndServe(Config.Port, nil))
}
