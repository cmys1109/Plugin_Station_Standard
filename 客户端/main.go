/*
BDS_Plugins_Manager
我个人给其定义是BDS服务端的一个插件包管理工具，我对包管理的了解尚浅，所以此程序的一些逻辑也不够成熟，是以最直观化的思维去解决的问题的

我将依赖分成两类，在定义变量名时并不是很严谨，故在此做出一些解释
依赖分为不同的level，分别命名为level大于等于2的称之为Plugin，level小于2的称之为Depend。level用作详细分割插件级别，当让Plugin也可以被作为依赖写在Details.json中的depends项内
level从小到大表示从底层到上层，但是暂时level作用并不是很多大，甚至有些冗余。但是创建它必定是有意义的，我认为可以以备后患，也可以为将来的发展做出一些铺垫，如果将来有发展的话
Plugin都按照相关规定存放于指定的一个github仓库内
由于gitee存在访问限制，所以只能无奈采用github，因为网络访问存在很大的影响，所以采用了访问镜像站来提高访问速度
并且提供了config.json来自定义访问的镜像站
据我观察，镜像站存在两种格式，一种是直接在镜像站域名后直接将接上带有http://的github地址，另一种是想GitHub域名替换为镜像站域名
所以在config.json中提供了两种访问模式，需要自行填写正确的模式
前者称之为splice，后者称之为parse，实在想不到什么好词了

Depend都定义于指定仓库内的Depend.json中，程序会先读取Depend.json中指定的地址，再按照读取到的地址通过githubAPI下载最新版发行版
*/
package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type (
	Version struct {
		MajorVersionNumber int
		MinorVersionNumber int
		RevisionNumber     int
	}
	//Version interface {
	//	ToString() string
	//	IsLastest(GetVer version) bool
	//}
)

// ToString 将version转化为string
func (version Version) ToString() string {
	var versionstr = "V "
	versionstr = versionstr + strconv.Itoa(version.MajorVersionNumber) + "."
	versionstr = versionstr + strconv.Itoa(version.MinorVersionNumber) + "."
	versionstr = versionstr + strconv.Itoa(version.RevisionNumber)
	return versionstr
}

func (version Version) IsLastest(GetVer Version) bool {
	if version.MajorVersionNumber < GetVer.MajorVersionNumber {
		return false
	} else if version.MinorVersionNumber < GetVer.MinorVersionNumber {
		return false
	} else if version.RevisionNumber < GetVer.RevisionNumber {
		return false
	}
	if version.MajorVersionNumber >= GetVer.MajorVersionNumber {
		return true
	}
	return true
}

func StringToVersion(PluginVer string) (Version, error) {
	var PluginSlice = strings.Split(PluginVer, ".")
	if len(PluginSlice) != 3 {
		return Version{}, errors.New("version type is not in GNU format")
	}
	minor, err := strconv.Atoi(PluginSlice[1])
	if err != nil {
		return Version{}, err
	}
	major, err := strconv.Atoi(strings.Split(PluginSlice[0], " ")[1])
	if err != nil {
		return Version{}, err
	}
	re, err := strconv.Atoi(PluginSlice[2])
	if err != nil {
		return Version{}, err
	}

	return Version{major, minor, re}, nil
}

type DecodeURL struct {
	URL      string `json:"url"`
	LinkMode string `json:"link_mode"`
}

type app struct {
	UserAgent    string               `json:"user_agent"`
	GetPluginURL map[string]DecodeURL `json:"get_plugin_url"`
	GetDependURL map[string]DecodeURL `json:"get_depend_url"`
}

func urlDecode(url DecodeURL, Addr string) string {
	urls := ""
	if url.LinkMode == "parse" {
		AddrSlice := strings.Split(Addr, "/")
		urls += url.URL
		for i := range AddrSlice {
			if i > 2 && i != len(AddrSlice)-1 && AddrSlice[i] != "blob" {
				urls += AddrSlice[i] + "/"
			} else if i == len(AddrSlice)-1 && AddrSlice[i] != "blob" {
				urls += AddrSlice[i]
			}
		}
	} else if url.LinkMode == "splice" {
		urls = url.URL + Addr
	} else {
		fmt.Println(errors.New("不支持的LinkMode"))
	}

	return urls
}

type Details struct {
	Pluginname string              `json:"pluginname"`
	Version    string              `json:"version"`
	Developer  string              `json:"developer"`
	Depends    map[string][]string `json:"depends"`
	Level      int                 `json:"level"`
	InstallCmd [][]string          `json:"install_cmd"`
	UpdateCmd  [][]string          `json:"update_cmd"`
}

type DependList struct {
	URL        string     `json:"url"`
	Level      int        `json:"level"`
	InstallCmd [][]string `json:"install_cmd"`
	UpdateCmd  [][]string `json:"update_cmd"`
}

//用于解析api.github
type (
	MyBody struct {
		Assets []Assets `json:"assets"`
	}
	Assets struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	}
)

var (
	DependDownloadURL DecodeURL
	PluginDownloadURL DecodeURL
	MakeAllDirList    []string
	App               app
)

func main() {
	_, err := os.Stat("./temp")
	if os.IsNotExist(err) {
		err := os.Mkdir("./temp", fs.ModePerm)
		if err != nil {
			return
		}
	}

	_, err = os.Stat("./BPM")
	if os.IsNotExist(err) {
		err := os.Mkdir("./BPM", fs.ModePerm)
		if err != nil {
			return
		}
	}

	_, err = os.Stat("./BPM/config.json")
	if os.IsNotExist(err) {
		ConfigFile, err := os.OpenFile("./BPM/config.json", os.O_RDWR|os.O_CREATE, 0766)
		if err != nil {
			fmt.Println(err.Error())
		}
		_, err = ConfigFile.Write([]byte("{\n  \"user_agent\": \"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36\",\n  \"get_plugin_url\": {\n    \"preferred\": {\n      \"url\": \"https://raw.fastgit.org/\",\n      \"link_mode\": \"parse\"\n    },\n    \"alternate\": {\n      \"url\": \"https://raw.githubusercontent.com/\",\n      \"link_mode\": \"parse\"\n    }\n  },\n  \"get_depend_url\": {\n    \"preferred\": {\n      \"url\": \"https://ghproxy.com/\",\n      \"link_mode\": \"splice\"\n    },\n    \"alternate\": {\n      \"url\": \"\",\n      \"link_mode\": \"\"\n    }\n  }\n}"))
		if err != nil {
			return
		}
		err = ConfigFile.Close()
		if err != nil {
			return
		}
	}

	var js, _ = ioutil.ReadFile("./BPM/config.json")
	var jsonerr = json.Unmarshal(js, &App)
	if jsonerr != nil {
		fmt.Println(jsonerr)
		var goin string
		scanln, err := fmt.Scanln(&goin)
		if err != nil {
			fmt.Println(scanln)
			return
		}
		return
	}

	_, err = os.Stat("./BPM/Depends.json")
	if os.IsNotExist(err) {
		DependJsonFile, err := os.OpenFile("./BPM/Depends.json", os.O_RDWR|os.O_CREATE, 0766)
		if err != nil {
			fmt.Println(err.Error())
		}
		_, err = DependJsonFile.Write([]byte("[]"))
		if err != nil {
			return
		}
		err = DependJsonFile.Close()
		if err != nil {
			return
		}
	}

	_, err = os.Stat("./BPM/PluginList.json")
	if os.IsNotExist(err) {
		PluginListFile, err := os.OpenFile("./BPM/PluginList.json", os.O_RDWR|os.O_CREATE, 0766)
		if err != nil {
			fmt.Println(err.Error())
		}
		_, err = PluginListFile.Write([]byte("{}"))
		if err != nil {
			return
		}
		err = PluginListFile.Close()
		if err != nil {
			return
		}
	}

	if !TryLink(App.GetPluginURL["preferred"]) {
		fmt.Println("PreferredURL outdated")
		fmt.Println("Change download address...")
		if !TryLink(App.GetPluginURL["alternate"]) {
			fmt.Println("AlternateURL outdated")
			fmt.Println("stop...")
		} else {
			PluginDownloadURL = App.GetPluginURL["alternate"]
		}
	} else {
		PluginDownloadURL = App.GetPluginURL["preferred"]
	}
	if !TryLink(App.GetDependURL["preferred"]) {
		fmt.Println("PreferredURL outdated")
		fmt.Println("Change download address...")
		if !TryLink(App.GetDependURL["alternate"]) {
			fmt.Println("AlternateURL outdated")
			fmt.Println("stop...")
		} else {
			DependDownloadURL = App.GetDependURL["alternate"]
		}
	} else {
		DependDownloadURL = App.GetDependURL["preferred"]
	}

	fmt.Println("Started!")
	var command, PluginKey string
	for true {
		var PluginList map[string]map[string]string
		PluginListByte, err := ioutil.ReadFile("./BPM/PluginList.json")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		err = json.Unmarshal(PluginListByte, &PluginList)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Print(">>>")
		_, _ = fmt.Scanln(&command, &PluginKey)

		switch command {
		case "install":
			err = Install(PluginKey)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		case "uninstall":
			err = UninstallPlugin(PluginKey)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		case "update":
			if PluginKey == "-a" {
				PluginListFile, err := ioutil.ReadFile("./BPM/PluginList.json")
				var PluginList map[string]map[string]string
				err = json.Unmarshal(PluginListFile, &PluginList)
				if err != nil {
					fmt.Println(err.Error())
					return
				}

				for i := range PluginList {
					err := UpdatePlugin(i)
					if err != nil {
						fmt.Println(err.Error())
						return
					}
				}
				fmt.Println("All plug-ins are up to date.")
			} else {
				err := UpdatePlugin(PluginKey)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
			}
		case "0":
			fmt.Println("Stop...")
			return
		case "depend": //临时接口，直接安装依赖
			err := InstallDepend(PluginKey)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}
	}

	return
}

// Install 调用GetDetails函数获取Plugin相应Details，解析并进行相应操作，
// 先下载所需依赖，并且对应依赖和插件调用响应函数进行安装
func Install(Plugin string) error {
	Details, err := GetDetails(Plugin)
	if err != nil {
		return err
	}

	if len(Details.Depends["depends"]) != 0 || len(Details.Depends["plugins"]) != 0 {
		if len(Details.Depends["depends"]) != 0 {
			DependByte, err := ioutil.ReadFile("./BPM/Depends.json")
			if err != nil {
				return err
			}
			var DependsList []string
			err = json.Unmarshal(DependByte, &DependsList)
			if err != nil {
				return err
			}

			for i, Depend := range Details.Depends["depends"] {
				for _, v := range DependsList {
					if v == Depend {
						Details.Depends["depends"] = append(Details.Depends["depends"][:i], Details.Depends["depends"][i+1:]...)
					}
				}
			}
		}

		if len(Details.Depends["plugins"]) != 0 {
			var DependsList map[string]map[string]string
			DependByte, err := ioutil.ReadFile("./BPM/PluginList.json")
			if err != nil {
				return err
			}
			err = json.Unmarshal(DependByte, &DependsList)
			if err != nil {
				return err
			}

			for i, Depend := range Details.Depends["plugins"] {
				for k := range DependsList {
					if k == Depend {
						Details.Depends["plugins"] = append(Details.Depends["plugins"][:i], Details.Depends["plugins"][i+1:]...)
					}
				}
			}
		}

		if len(Details.Depends["depends"]) != 0 || len(Details.Depends["plugins"]) != 0 {
			fmt.Println("需要安装以下依赖：")
			i := 1
			for _, v := range Details.Depends["depends"] {
				fmt.Print(i, ".", v, "  ")
				i++
			}
			for _, v := range Details.Depends["plugins"] {
				fmt.Print(i, ".", v, "  ")
				i++
			}
			fmt.Print("\n")
			for Details.Depends["depends"] != nil || Details.Depends["plugins"] != nil {
				fmt.Print("键入以继续安装(Yes/No)>>>")
				var c string
				_, _ = fmt.Scanln(&c)
				if c == "Yes" || c == "Y" || c == "y" || c == "YES" || c == "yes" {
					fmt.Println("start install...")
					break
				} else if c == "NO" || c == "No" || c == "N" || c == "n" {
					fmt.Println("stop install...")
					return nil
				} else {
					fmt.Println("ERR command")
				}
			}

			for _, Depend := range Details.Depends["depends"] {
				err := InstallDepend(Depend)
				if err != nil {
					return err
				}
			}
			for _, Depend := range Details.Depends["plugins"] {
				err := Install(Depend)
				if err != nil {
					return err
				}
			}
		}
	}

	err = GetPlugin(Plugin)
	if err != nil {
		return err
	}

	err = InstallPlugin(Details, Plugin)
	if err != nil {
		return err
	}
	fmt.Println(Plugin, " Successfully installed!")
	return nil
}

// TryLink 尝试连接给定的下载URL，如果可用存为DownloadURL
func TryLink(url DecodeURL) bool {
	fmt.Println("Try to linking...")
	cl := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequest("GET", urlDecode(url, "https://github.com/cmys1109/Plugin-Station/blob/main/README.md"), nil)
	req.Header.Set("User-Agent", App.UserAgent)
	res, err := cl.Do(req)
	if err != nil {
		return false
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

	if res.StatusCode == 200 {
		fmt.Println("StatusCode = 200,Really to download...")
		return true
	} else {
		fmt.Println("StatusCode Error :" + strconv.Itoa(res.StatusCode))
		return false
	}
}

// GetPlugin 下载Plugin本体文件，存放于./temp目录内
func GetPlugin(PluginKey string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	req, _ := http.NewRequest("GET", urlDecode(PluginDownloadURL, "https://raw.githubusercontent.com/cmys1109/Plugin-Station/main/Plugins/"+PluginKey), nil)
	req.Header.Set("User-Agent", App.UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}

	}(resp.Body)

	if resp.StatusCode == 200 {
		f, err := os.Create("./temp/" + PluginKey)
		if err != nil {
			return err
		}
		fmt.Println("Downloading...")
		StarTime := time.Now().UnixNano()
		written, err := io.Copy(f, resp.Body)
		if err != nil {
			fmt.Println(written)
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
		EndTime := time.Now().UnixNano()
		seconds := float64((EndTime - StarTime) / 1e9)
		fmt.Print("The download took ")
		fmt.Print(seconds)
		fmt.Println(" seconds.")
	} else {
		fmt.Println("url link err" + strconv.Itoa(resp.StatusCode))
		return errors.New("StatusCode:" + strconv.Itoa(resp.StatusCode))
	}
	return nil
}

// GetDetails 读取Plugin对应的Detail.json，传回Detail
func GetDetails(PluginKey string) (Details, error) {
	cl := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", urlDecode(PluginDownloadURL, "https://raw.githubusercontent.com/cmys1109/Plugin-Station/main/Details/"+strings.Split(PluginKey, ".")[0]+".json"), nil)
	req.Header.Set("User-Agent", App.UserAgent)
	resp, err := cl.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		var ReturnDetails Details
		return ReturnDetails, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	body, err := ioutil.ReadAll(resp.Body)
	var details Details
	jsonerr := json.Unmarshal(body, &details)
	if jsonerr != nil {
		return details, jsonerr
	}
	return details, nil
}

// InstallPlugin 读取InstallCmd，并且遍历并进行操作，最后将Plugin相应信息写入PluginList.json
func InstallPlugin(details Details, PluginKey string) error {
	var PluginList map[string]map[string]string
	PluginListByte, err := ioutil.ReadFile("./BPM/PluginList.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(PluginListByte, &PluginList)
	if err != nil {
		return err
	}
	if PluginList[PluginKey] != nil {
		fmt.Println(PluginKey, " installed.")
		return nil
	}
	_, err = StringToVersion(details.Version)
	if err != nil {
		return err
	}
	fmt.Println(details.Pluginname, " start to install...")
	fmt.Println("Plugin version:", details.Version)
	fmt.Println("Plugin developer:", details.Developer)
	PluginListStruct := make(map[string]string)
	PluginListStruct["version"] = details.Version
	for Com, Command := range details.InstallCmd {
		fmt.Println("Run command:", Com)
		switch Command[0] {
		case "unzip":
			err := Unzip(Command[1], Command[2])
			if err != nil {
				return err
			}
		case "copy":
			err := CopyFile(Command[1], Command[2])
			if err != nil {
				return err
			}
			PluginListStruct["file_"+Command[2]] = Command[2]
		case "del":
			err := DelFileOrDir(Command[1])
			if err != nil {
				return err
			}
		}
	}

	PluginList[PluginKey] = PluginListStruct
	PluginListByte, err = json.Marshal(PluginList)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("./BPM/PluginList.json", PluginListByte, fs.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// UpdatePlugin 读取仓库内对应Detail.json判断是否为最新版
//如果是调用GetPlugin下载最新版，并且读取UpdateCmd进行相应操作
func UpdatePlugin(PluginKey string) error {
	PluginListFile, err := ioutil.ReadFile("./BPM/PluginList.json")
	var PluginList map[string]map[string]string
	err = json.Unmarshal(PluginListFile, &PluginList)
	if err != nil {
		return err
	}
	NowVersionStr := PluginList[PluginKey]["version"]
	NowVersion, err := StringToVersion(NowVersionStr)
	if err != nil {
		return err
	}

	details, err := GetDetails(PluginKey)
	if err != nil {
		return err
	}
	NewVersion, err := StringToVersion(details.Version)
	if err != nil {
		return err
	}

	if NowVersion.IsLastest(NewVersion) {
		fmt.Println(PluginKey, " is lastest")
		return nil
	}
	err = GetPlugin(PluginKey)
	if err != nil {
		return err
	}
	var PluginListStruct map[string]string
	PluginListStruct = PluginList[PluginKey]
	for Com, Command := range details.UpdateCmd {
		fmt.Println("Run command:", Com)
		switch Command[0] {
		case "unzip":
			err := Unzip(Command[1], Command[2])
			if err != nil {
				return err
			}
		case "copy":
			err := CopyFile(Command[1], Command[2])
			if err != nil {
				return err
			}
			PluginListStruct["file_"+Command[2]] = Command[2]
		case "del":
			err := DelFileOrDir(Command[1])
			if err != nil {
				return err
			}
			delete(PluginListStruct, "file_ "+Command[1])
		}
	}
	PluginList[PluginKey] = PluginListStruct
	PluginListByte, err := json.Marshal(PluginList)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("./BPM/PluginList.json", PluginListByte, fs.ModePerm)
	if err != nil {
		return err
	}
	fmt.Println(PluginKey, " updated")

	return nil
}

// UninstallPlugin 读取PluginList.json内相应内容，并且删除所记录的文件以及json内相应内容
func UninstallPlugin(PluginKey string) error {
	PluginListFile, err := ioutil.ReadFile("./BPM/PluginList.json")
	if err != nil {
		return err
	}

	var PluginList map[string]map[string]string
	err = json.Unmarshal(PluginListFile, &PluginList)
	if err != nil {
		return err
	}

	FileList := PluginList[PluginKey]
	if FileList == nil {
		fmt.Println("'" + PluginKey + "'" + "not installed.")
		return nil
	}
	for i := range FileList {
		if i != "version" {
			err = DelFileOrDir(FileList[i])
			if err != nil {
				return err
			}
		}

	}

	delete(PluginList, PluginKey)
	PluginListByte, err := json.Marshal(PluginList)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("./BPM/PluginList.json", PluginListByte, fs.ModePerm)
	if err != nil {
		return err
	}
	fmt.Println(PluginKey, " uninstalled.")

	return nil
}

// InstallDepend 涵盖依赖的下载安装，基本方法与Plugin相同
//不同之处在于，获取依赖通过读取仓库内Depends.json进行对应URL下载
func InstallDepend(Depend string) error {
	DependByte, err := ioutil.ReadFile("./BPM/Depends.json")
	if err != nil {
		return err
	}
	var DependsList []string
	err = json.Unmarshal(DependByte, &DependsList)
	if err != nil {
		return err
	}
	//var DependList
	for _, v := range DependsList {
		if v == Depend {
			fmt.Println(Depend + " installed")
			return nil
		}
	}
	cl := &http.Client{Timeout: 120 * time.Second}
	request, _ := http.NewRequest("GET", urlDecode(DependDownloadURL, "https://raw.githubusercontent.com/cmys1109/Plugin-Station/main/Depends.json"), nil)
	request.Header.Set("User-Agent", App.UserAgent)
	resqde, err := cl.Do(request)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resqde.Body)
	DependBody, err := ioutil.ReadAll(resqde.Body)
	if err != nil {
		return err
	}
	var Depends map[string]DependList
	err = json.Unmarshal(DependBody, &Depends)
	if err != nil {
		return err
	}
	var exist = false
	for key := range Depends {
		_, DependExist := Depends[key]
		if DependExist == true {
			exist = true
			break
		}
	}
	if exist == false {
		return errors.New("the depend non-existent")
	}

	//
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+Depends[Depend].URL+"/releases/latest", nil)
	req.Header.Set("User-Agent", App.UserAgent)
	resq, err := cl.Do(req)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}

	}(resq.Body)

	body, _ := ioutil.ReadAll(resq.Body)
	if err != nil {
		return err
	}

	var xxm MyBody
	err = json.Unmarshal(body, &xxm)
	if err != nil {
		return err
	}

	BrowserDownloadURL := xxm.Assets[0].BrowserDownloadURL //最新发行版下载地址

	//下载
	r, _ := http.NewRequest("GET", urlDecode(DependDownloadURL, BrowserDownloadURL), nil)
	r.Header.Set("User-Agent", App.UserAgent)
	re, err := cl.Do(r)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}

	}(re.Body)

	if re.StatusCode == 200 {
		f, err := os.Create("./temp/" + strings.Split(BrowserDownloadURL, "/")[(len(strings.Split(BrowserDownloadURL, "/"))-1)])
		if err != nil {
			return err
		}
		fmt.Println("Downloading...")
		StarTime := time.Now().UnixNano()
		written, err := io.Copy(f, re.Body)
		if err != nil {
			fmt.Println(written)
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
		EndTime := time.Now().UnixNano()
		seconds := float64((EndTime - StarTime) / 1e9)
		fmt.Print("The download took ")
		fmt.Print(seconds)
		fmt.Println(" seconds.")
	} else {
		fmt.Println("url link err" + strconv.Itoa(re.StatusCode))
		return errors.New("StatusCode:" + strconv.Itoa(re.StatusCode))
	}

	//安装
	for _, v := range Depends[Depend].InstallCmd {
		Command := v
		switch Command[0] {
		case "unzip":
			err := Unzip(Command[1], Command[2])
			if err != nil {
				return err
			}
			break
		case "copy":
			err := CopyFile(Command[1], Command[2])
			if err != nil {
				return err
			}
		case "copydir":
			err := CopyDir(Command[1], Command[2])
			if err != nil {
				return err
			}
		case "del":
			err := DelFileOrDir(Command[1])
			if err != nil {
				return err
			}
		}
	}

	DependByte, err = ioutil.ReadFile("./BPM/Depends.json")
	if err != nil {
		return err
	}
	var DependList []string
	err = json.Unmarshal(DependByte, &DependList)
	if err != nil {
		return err
	}
	DependList = append(DependList, Depend)

	DependByte, err = json.Marshal(DependList)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("./BPM/Depends.json", DependByte, fs.ModePerm)
	if err != nil {
		return err
	}

	fmt.Println(Depend, " installed")
	return nil
}

// Unzip 解压zip至指定目录
func Unzip(zipFile string, destDir string) error { //https://www.jianshu.com/p/4593cfffb9e9
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer func(zipReader *zip.ReadCloser) {
		err := zipReader.Close()
		if err != nil {

		}
	}(zipReader)

	for _, f := range zipReader.File {
		FilePath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(FilePath, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			if err = os.MkdirAll(filepath.Dir(FilePath), os.ModePerm); err != nil {
				return err
			}

			inFile, err := f.Open()
			if err != nil {
				return err
			}
			defer func(inFile io.ReadCloser) {
				err := inFile.Close()
				if err != nil {

				}
			}(inFile)

			outFile, err := os.OpenFile(FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func(outFile *os.File) {
				err := outFile.Close()
				if err != nil {

				}
			}(outFile)

			_, err = io.Copy(outFile, inFile)
			if err != nil {
				return err
			}
			//inFile.Close()
			//outFile.Close()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// DirTreetop 采用递归方式取得目录树树梢
// 目录树树梢顾名思义，是最深处的子目录
func DirTreetop(Dir string) error {
	dir, err := ioutil.ReadDir(Dir)
	if err != nil {
		fmt.Println(err.Error())
	}
	var Have = false
	var DirList []string
	for i := range dir {
		if dir[i].IsDir() {
			Have = true
			DirList = append(DirList, dir[i].Name())
		}
	}

	if Have {
		for i := range DirList {
			err := DirTreetop(Dir + "\\" + DirList[i])
			if err != nil {
				return err
			}
		}
	} else {
		MakeAllDirList = append(MakeAllDirList, Dir)
	}

	return nil
}

// CopyDir 通过DirTreeTop函数读取目录树树梢，并通过os.MkdirAll()快速建立目录树，再将所有文件复制进相应目录
func CopyDir(SrcDir, DestDir string) error {
	var DestDirList []string
	MakeAllDirList = nil
	err := DirTreetop(SrcDir) //读取目录树树梢
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	for _, paths := range MakeAllDirList { //创建目录树中所有目录
		rel, err := filepath.Rel(SrcDir, paths)
		if err != nil {
			return err
		}
		DestDirList = append(DestDirList, DestDir+"\\"+rel)
		err = os.MkdirAll(DestDir+"\\"+rel, fs.ModePerm)
		if err != nil {
			return err
		}
	}

	err = filepath.Walk(SrcDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err.Error())
			return err
		}

		if !info.IsDir() {
			rel, err := filepath.Rel(SrcDir, path)
			if err != nil {
				return err
			}
			err = CopyFile(path, DestDir+"\\"+rel)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// CopyFile 复制文件
func CopyFile(srcFile, destFile string) error {
	file1, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	file2, err := os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		if strings.Split(err.Error(), ":")[1] == " The system cannot find the path specified." {
			var InstallDir = ""
			for i := 0; i < len(strings.Split(destFile, "/"))-2; i++ {
				InstallDir = InstallDir + strings.Split(destFile, "/")[i] + "/"
			}
			InstallDir = InstallDir + strings.Split(destFile, "/")[len(strings.Split(destFile, "/"))-2]
			err = os.MkdirAll(InstallDir, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	file2, err = os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil
	}
	_, err = io.Copy(file2, file1)
	if err != nil {
		return err
	}
	err = file1.Close()
	if err != nil {
		return err
	}
	err = file2.Close()
	if err != nil {
		return err
	}

	return nil
}

// DelFileOrDir 判断是目录还是文件，并且进行相应的删除操作
func DelFileOrDir(name string) error {
	info, err := os.Stat(name)
	if err != nil {
		return err
	}
	if info.IsDir() {
		err := os.RemoveAll(name)
		if err != nil {
			return err
		}
	} else {
		err := os.Remove(name)
		if err != nil {
			return err
		}
	}

	return nil
}
