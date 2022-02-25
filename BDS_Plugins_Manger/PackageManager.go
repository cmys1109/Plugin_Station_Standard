package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

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
		Logger(1, "Downloading...")
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
		Logger(1, "The download took "+strconv.FormatFloat(seconds, 'E', 1, 64)+" seconds.")
	} else {
		fmt.Println("url link err" + strconv.Itoa(resp.StatusCode))
		return errors.New("StatusCode:" + strconv.Itoa(resp.StatusCode))
	}

	return nil
}

// InstallPlugin 读取InstallCmd，并且遍历并进行操作，最后将Plugin相应信息写入PluginList.json
func InstallPlugin(details Details, PluginKey string) error {
	var PluginList map[string]PluginLog
	PluginListByte, err := ioutil.ReadFile("./BPM/PluginList.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(PluginListByte, &PluginList)
	if err != nil {
		return err
	}
	for k := range PluginList {
		if k == PluginKey {
			Logger(1, PluginKey+" installed.")
			return nil
		}
	}

	_, err = StringToVersion(details.Version)
	if err != nil {
		return err
	}
	Logger(2, details.Pluginname+" start to install...\n"+"Plugin version:"+details.Version)
	PluginListStruct := make(map[string]string)
	PluginListStruct["version"] = details.Version
	FileLogSlice, err := CmdCore(details.InstallCmd, PluginList[PluginKey].File)
	if err != nil {
		return err
	}
	var PluginLogSCR PluginLog
	for _, v := range FileLogSlice {
		PluginLogSCR.File = append(PluginLogSCR.File, v)
	}
	PluginLogSCR.Version = details.Version
	PluginList[PluginKey] = PluginLogSCR

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
	var PluginList map[string]PluginLog
	err = json.Unmarshal(PluginListFile, &PluginList)
	if err != nil {
		return err
	}
	NowVersionStr := PluginList[PluginKey].Version
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
		Logger(1, PluginKey+" is lastest")
		return nil
	}
	err = GetPlugin(PluginKey)
	if err != nil {
		return err
	}
	FileLogSlice, err := CmdCore(details.InstallCmd, PluginList[PluginKey].File)
	if err != nil {
		return err
	}
	var PluginLogSCR PluginLog
	for _, v := range FileLogSlice {
		PluginLogSCR.File = append(PluginLogSCR.File, v)
	}
	PluginLogSCR.Version = details.Version
	PluginList[PluginKey] = PluginLogSCR
	PluginListByte, err := json.Marshal(PluginList)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("./BPM/PluginList.json", PluginListByte, fs.ModePerm)
	if err != nil {
		return err
	}
	Logger(2, PluginKey+" updated.Now version "+NewVersion.ToString())

	return nil
}

// UninstallPlugin 读取PluginList.json内相应内容，并且删除所记录的文件以及json内相应内容
func UninstallPlugin(PluginKey string) error {
	PluginListFile, err := ioutil.ReadFile("./BPM/PluginList.json")
	if err != nil {
		return err
	}

	var PluginList map[string]PluginLog
	err = json.Unmarshal(PluginListFile, &PluginList)
	if err != nil {
		return err
	}

	FileList := PluginList[PluginKey].File
	if FileList == nil {
		Logger(1, "'"+PluginKey+"'"+"not installed.")
		return nil
	}
	for i := range FileList {
		err = DelFileOrDir(FileList[i])
		if err != nil {
			return err
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
	Logger(2, PluginKey+" uninstalled.")

	return nil
}

// InstallDepend 涵盖依赖的下载安装，基本方法与Plugin相同
//不同之处在于，获取依赖通过读取仓库内Depends.json进行对应URL下载
func InstallDepend(Depend string) error {
	DependByte, err := ioutil.ReadFile("./BPM/Depends.json")
	if err != nil {
		return err
	}
	var DependsList map[string]PluginLog
	err = json.Unmarshal(DependByte, &DependsList)
	if err != nil {
		return err
	}
	//var DependList
	for k := range DependsList {
		if k == Depend {
			Logger(1, Depend+" installed")
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
		if key == Depend {
			exist = true
			break
		}
	}
	if exist == false {
		Logger(4, "the depend <"+Depend+"> non-existent")
		return nil
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

	if len(xxm.Assets) == 0 {
		Logger(4, "Not found <"+Depend+">'s browser_download_url\nContact the developer to resolve\ndetails:<"+"https://api.github.com/repos/"+Depends[Depend].URL+"/releases/latest>")
		return nil
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
		Logger(1, "Downloading...")
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
		Logger(1, "The download took "+strconv.FormatFloat(seconds, 'E', 1, 64)+" seconds.")
	} else {
		fmt.Println("url link err" + strconv.Itoa(re.StatusCode))
		return errors.New("StatusCode:" + strconv.Itoa(re.StatusCode))
	}

	//安装
	FileLogSlice, err := CmdCore(Depends[Depend].InstallCmd, nil)
	if err != nil {
		return err
	}

	var DependLogSCR PluginLog
	DependLogSCR.File = FileLogSlice
	BrowserDownloadURLSlice := strings.Split(BrowserDownloadURL, "/")
	DependLogSCR.Version = BrowserDownloadURLSlice[len(BrowserDownloadURLSlice)-2]
	DependsList[Depend] = DependLogSCR

	DependByte, err = json.Marshal(DependsList)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("./BPM/Depends.json", DependByte, fs.ModePerm)
	if err != nil {
		return err
	}

	Logger(2, Depend+" installed")
	return nil
}

// UnInstallDepend 卸载依赖函数
//
// 完全复制UnInstallPlugin函数，仅仅修改了函数名和调用文件
// 如无必要，勿增实体,复制粘贴，方便易用
func UnInstallDepend(PluginKey string) error {
	PluginListFile, err := ioutil.ReadFile("./BPM/Depends.json")
	if err != nil {
		return err
	}

	var PluginList map[string]PluginLog
	err = json.Unmarshal(PluginListFile, &PluginList)
	if err != nil {
		return err
	}

	has := false
	for k := range PluginList {
		if k == PluginKey {
			has = true
			break
		}
	}
	if has == false {
		Logger(1, "'"+PluginKey+"'"+"not installed.")
		return nil
	}
	FileList := PluginList[PluginKey].File
	for i := range FileList {
		err = DelFileOrDir(FileList[i])
		if err != nil {
			return err
		}
	}

	delete(PluginList, PluginKey)
	PluginListByte, err := json.Marshal(PluginList)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("./BPM/Depends.json", PluginListByte, fs.ModePerm)
	if err != nil {
		return err
	}
	Logger(2, PluginKey+" uninstalled.")

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

// OutPluginsList 输出Plugin列表
func OutPluginsList() error {
	PluginListFile, err := ioutil.ReadFile("./BPM/PluginList.json")
	if err != nil {
		return err
	}
	var PluginList map[string]PluginLog
	err = json.Unmarshal(PluginListFile, &PluginList)
	if err != nil {
		return err
	}
	var i = 1
	fmt.Println("Plugins:")
	for k, v := range PluginList {
		fmt.Print(i)
		fmt.Println(".", k, "  ", v.Version)
		i++
	}
	if i == 1 {
		fmt.Println("<nil>")
	}

	DependListFile, err := ioutil.ReadFile("./BPM/Depends.json")
	if err != nil {
		return err
	}
	var DependList map[string]PluginLog
	err = json.Unmarshal(DependListFile, &DependList)
	i = 1
	fmt.Println("Depends:")
	for k, v := range DependList {
		fmt.Print(i)
		fmt.Println(".", k, "  ", v.Version)
		i++
	}
	if i == 1 {
		fmt.Println("<nil>")
	}

	return nil
}

// CreatPackage New一个包
func CreatPackage(PackageName string) error {
	//exist and creat package.json
	var PackJson PackJson
	_, err := os.Stat("./package.json")
	if os.IsNotExist(err) {
		file, err := os.OpenFile("./package.json", os.O_RDWR|os.O_CREATE, os.ModePerm)
		if err != nil {
			return err
		}
		err = file.Close()
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		Logger(4, "had creat the package")
		return nil
	}

	//write PackJson
	if PackageName != "" {
		PackJson.PackageName = PackageName
	}
	PackJson.Level = 2
	PackJson.PackageMap = GetPathMap(".")

	PackageByte, err := json.Marshal(PackJson)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("./package.json", PackageByte, os.ModePerm)
	if err != nil {
		return err
	}

	Logger(2, "成功创建 "+PackageName)
	return nil
}

// InstallPackage 安装包
func InstallPackage(Pack string) error {
	if TryLink(DecodeURL{
		URL:      CmdToURL(Pack) + "/Package.json",
		LinkMode: "0",
	}) == false {
		Logger(3, "ERR:"+Pack+" cannot be identified.")
		return nil
	}

	PackageJson, err := GetPackage(Pack)
	if err != nil {
		return err
	}

	for _, v := range PackageJson.PackageMap["Dir"] {
		err := os.MkdirAll(v, os.ModePerm)
		if err != nil {
			return err
		}
	}

	for _, v := range PackageJson.PackageMap["File"] {
		err := CopyFile(".\\temp\\"+PackageJson.PackageName+"\\"+v, ".\\"+v)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetPackage 下载包并解压传出PackageJson
func GetPackage(PackageURL string) (PackJson, error) {
	client := &http.Client{Timeout: 60 * time.Second} //github.com/cmys1109/Plugin-Station@master
	req, _ := http.NewRequest("GET", ZipDownloadURL(PackageURL), nil)
	req.Header.Set("User-Agent", App.UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return PackJson{}, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}

	}(resp.Body)

	var urlSlice []string
	var ZipFile string

	if resp.StatusCode == 200 {
		urlSlice = strings.Split(strings.Split(PackageURL, "@")[0], "/")
		ZipFile = "./temp/" + urlSlice[len(urlSlice)-1] + ".zip"
		f, err := os.Create(ZipFile) //取连接最后一个单位为包名，即为仓库名
		if err != nil {
			return PackJson{}, err
		}
		Logger(1, "Downloading...")
		StarTime := time.Now().UnixNano()
		written, err := io.Copy(f, resp.Body)
		if err != nil {
			fmt.Println(written)
			return PackJson{}, err
		}
		err = f.Close()
		if err != nil {
			return PackJson{}, err
		}
		EndTime := time.Now().UnixNano()
		seconds := float64((EndTime - StarTime) / 1e9)
		Logger(1, "The download took "+strconv.FormatFloat(seconds, 'E', 1, 64)+" seconds.")
	} else {
		fmt.Println("url link err" + strconv.Itoa(resp.StatusCode))
		return PackJson{}, errors.New("StatusCode:" + strconv.Itoa(resp.StatusCode))
	}

	err = Unzip(ZipFile, "./temp/"+urlSlice[len(urlSlice)-1])
	if err != nil {
		return PackJson{}, err
	}

	PackageByte, err := ioutil.ReadFile("./temp/" + urlSlice[len(urlSlice)-1] + "/" + "package.json")
	if err != nil {
		return PackJson{}, err
	}
	var PackageJson PackJson
	err = json.Unmarshal(PackageByte, &PackageJson)
	if err != nil {
		return PackageJson, err
	}

	return PackageJson, nil
}

func CmdToURL(PackageURL string) string {
	var URL string

	if PluginDownloadURL.LinkMode == "splice" {
		URL = urlDecode(PluginDownloadURL, "https://"+PackageURL)
	} else if DependDownloadURL.LinkMode == "splice" {
		URL = urlDecode(DependDownloadURL, "https://"+PackageURL)
	} else {
		URL = "https://ghproxy.com/https://" + PackageURL
	}

	return URL
}

func ZipDownloadURL(PackageURL string) string {
	URLSlice := strings.Split(PackageURL, "@")
	if len(URLSlice) != 2 {
		return "err"
	}
	return CmdToURL(URLSlice[0]) + "/archive/refs/heads/" + URLSlice[1] + ".zip"
}
