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
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

func main() {
	DependDownloadURL = DecodeURL{URL: "https://ghproxy.com/", LinkMode: "splice"}
	PluginDownloadURL = DependDownloadURL

	_, err := os.Stat("./temp")
	if os.IsNotExist(err) {
		err := os.Mkdir("./temp", fs.ModePerm)
		if err != nil {
			return
		}
	} else if err == nil {
		err := os.RemoveAll("./temp")
		if err != nil {
			return
		}
		err = os.Mkdir("./temp", fs.ModePerm)
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
			Logger(3, err.Error())
		}
		_, err = ConfigFile.Write([]byte("{\n  \"try_link\": false,\n  \"debug\": true,\n  \"user_agent\": \"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36\",\n  \"get_plugin_url\": {\n    \"preferred\": {\n      \"url\": \"https://ghproxy.com/\",\n      \"link_mode\": \"splice\"\n    },\n    \"alternate\": {\n      \"url\": \"https://raw.githubusercontent.com/\",\n      \"link_mode\": \"parse\"\n    }\n  },\n  \"get_depend_url\": {\n    \"preferred\": {\n      \"url\": \"https://ghproxy.com/\",\n      \"link_mode\": \"splice\"\n    },\n    \"alternate\": {\n      \"url\": \"\",\n      \"link_mode\": \"\"\n    }\n  }\n}"))
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
		Logger(3, jsonerr.Error())
		var goin string
		_, err := fmt.Scanln(&goin)
		if err != nil {
			Logger(3, err.Error())
			return
		}
		return
	}

	_, err = os.Stat("./BPM/Depends.json")
	if os.IsNotExist(err) {
		DependJsonFile, err := os.OpenFile("./BPM/Depends.json", os.O_RDWR|os.O_CREATE, 0766)
		if err != nil {
			Logger(3, err.Error())
		}
		_, err = DependJsonFile.Write([]byte("{}"))
		if err != nil {
			return
		}
		err = DependJsonFile.Close()
		if err != nil {
			return
		}
	}

	_, err = os.Stat("./BPM/Log")
	if os.IsNotExist(err) {
		LogFile, err := os.OpenFile("./BPM/Log", os.O_RDWR|os.O_CREATE, 0766)
		if err != nil {
			Logger(3, err.Error())
		}
		_, err = LogFile.Write([]byte("//BDS_Plugins_Manager Log\n"))
		if err != nil {
			return
		}
		err = LogFile.Close()
		if err != nil {
			return
		}
	}

	_, err = os.Stat("./BPM/PluginList.json")
	if os.IsNotExist(err) {
		PluginListFile, err := os.OpenFile("./BPM/PluginList.json", os.O_RDWR|os.O_CREATE, 0766)
		if err != nil {
			Logger(3, err.Error())
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

	if App.TryLink == true {
		var wgTL sync.WaitGroup
		wgTL.Add(2)
		go func() {
			if !TryLink(App.GetPluginURL["preferred"]) {
				Logger(4, "[GetPlugin]PreferredURL outdated")
				Logger(1, "[GetPlugin]Change download address...")
				if !TryLink(App.GetPluginURL["alternate"]) {
					Logger(4, "[GetPlugin]AlternateURL outdated")
				} else {
					PluginDownloadURL = App.GetPluginURL["alternate"]
				}
			} else {
				PluginDownloadURL = App.GetPluginURL["preferred"]
			}
			Logger(2, "[GetPlugin] "+PluginDownloadURL.URL)
			wgTL.Done()
		}()

		go func() {
			time.Sleep(100 * time.Millisecond)
			if !TryLink(App.GetDependURL["preferred"]) {
				Logger(4, "[GetDepend]PreferredURL outdated")
				Logger(1, "[GetDepend]Change download address...")
				if !TryLink(App.GetDependURL["alternate"]) {
					Logger(4, "[GetDepend]AlternateURL outdated")
				} else {
					DependDownloadURL = App.GetDependURL["alternate"]
				}
			} else {
				DependDownloadURL = App.GetDependURL["preferred"]
			}
			Logger(2, "[GetDepend] "+DependDownloadURL.URL)
			wgTL.Done()
		}()

		wgTL.Wait() //使用goroutine并行TryLink
	}

	Logger(2, "Started!")
	var command, PluginKey string

	for true {
		var PluginList map[string]PluginLog
		PluginListByte, err := ioutil.ReadFile("./BPM/PluginList.json")
		if err != nil {
			Logger(3, err.Error())
			return
		}
		err = json.Unmarshal(PluginListByte, &PluginList)
		if err != nil {
			Logger(3, err.Error())
			return
		}

		fmt.Print(">>>")
		_, _ = fmt.Scanln(&command, &PluginKey)

		switch command {
		case "install":
			err = Install(PluginKey)
			if err != nil {
				Logger(3, err.Error())
				return
			}
		case "uninstall":
			err = UninstallPlugin(PluginKey)
			if err != nil {
				Logger(3, err.Error())
				return
			}
		case "update":
			if PluginKey == "-a" {
				PluginListFile, err := ioutil.ReadFile("./BPM/PluginList.json")
				var PluginList map[string]PluginLog
				err = json.Unmarshal(PluginListFile, &PluginList)
				if err != nil {
					Logger(3, err.Error())
					return
				}

				for i := range PluginList {
					err := UpdatePlugin(i)
					if err != nil {
						Logger(3, err.Error())
						return
					}
				}
				Logger(2, "All plug-ins are up to date.")
			} else {
				err := UpdatePlugin(PluginKey)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
			}
		case "0":
			if App.Debug == true {
				//Debug模式下进行结束操作
				err = DebugFinisher()
				if err != nil {
					Logger(3, err.Error())
				} else {
					Logger(1, "DebugFinisher worked")
				}
			}
			Logger(2, "Stop...")
			return
		case "depend":
			err := InstallDepend(PluginKey)
			if err != nil {
				Logger(3, err.Error())
				return
			}
		case "list":
			err := OutPluginsList()
			if err != nil {
				Logger(3, err.Error())
				return
			}
		case "undepend":
			err := UnInstallDepend(PluginKey)
			if err != nil {
				Logger(3, err.Error())
				return
			}
		case "-c":
			err := CreatPackage(PluginKey)
			if err != nil {
				Logger(3, err.Error())
				return
			}
		case "-g":
			err := InstallPackage(PluginKey)
			if err != nil {
				Logger(3, err.Error())
				return
			}
		}
	}
	return
}

// Install 调用GetDetails函数获取Plugin相应Details，解析并进行相应操作，
// 先下载所需依赖，并且对应依赖和插件调用响应函数进行安装
func Install(Plugin string) error {
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
		if k == Plugin {
			Logger(1, Plugin+" installed.")
			return nil
		}
	}

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
			var DependsList map[string]PluginLog
			err = json.Unmarshal(DependByte, &DependsList)
			if err != nil {
				return err
			}

			for i, Depend := range Details.Depends["depends"] {
				for k := range DependsList {
					if k == Depend {
						Details.Depends["depends"] = append(Details.Depends["depends"][:i], Details.Depends["depends"][i+1:]...)
					}
				}
			}
		}

		if len(Details.Depends["plugins"]) != 0 {
			var DependsList map[string]PluginLog
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
				fmt.Print(1, "键入以继续安装(Yes/No)>>>")
				var c string
				_, _ = fmt.Scanln(&c)
				if c == "Yes" || c == "Y" || c == "y" || c == "YES" || c == "yes" {
					Logger(2, "start install...")
					break
				} else if c == "NO" || c == "No" || c == "N" || c == "n" {
					Logger(2, "stop install...")
					return nil
				} else {
					Logger(1, "ERR command")
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
	Logger(2, Plugin+" Successfully installed!")
	return nil
}

// TryLink 尝试连接给定的下载URL，如果可用存为DownloadURL
func TryLink(url DecodeURL) bool {
	Logger(1, "Try to linking...")
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
		Logger(1, "StatusCode = 200,URL:"+url.URL)
		return true
	} else {
		Logger(3, "StatusCode Error :"+strconv.Itoa(res.StatusCode))
		return false
	}
}

// CmdCore 统一的运行cmd的函数
func CmdCore(CmdSlice [][]string, FileLog []string) ([]string, error) {
	for _, v := range CmdSlice {
		Command := v
		switch Command[0] {
		case "unzip":
			err := Unzip(Command[1], Command[2])
			if err != nil {
				return FileLog, err
			}
			FileLog = append(FileLog, Command[2])
		case "copy":
			err := CopyFile(Command[1], Command[2])
			if err != nil {
				return FileLog, err
			}
			FileLog = append(FileLog, Command[2])
		case "copydir":
			err, FileLog := CopyDir(Command[1], Command[2])
			if err != nil {
				return FileLog, err
			}
			return FileLog, nil
		case "del":
			has := false
			for i := range FileLog {
				if FileLog[i] != Command[1] {
					continue
				}
				has = true
				err := DelFileOrDir(Command[1])
				if err != nil {
					return FileLog, err
				}
				FileLog = append(FileLog[:i], FileLog[i+1:]...)
				break
			}
			if has == false {
				Logger(4, Command[1]+"不在FileLog中，无权限删除")
			}
		case "syscmd":
			Logger(4, "请注意!正在调用SYSCMD  调用内容: "+Command[1])
			CharPtr, err := syscall.BytePtrFromString(Command[1])
			if err != nil {
				return FileLog, err
			}
			_, _, err = SYSCmd.Call(uintptr(unsafe.Pointer(CharPtr)))

		}
	}

	return FileLog, nil
}

// Logger 统一记录以及输出函数
/*

Type
     1 info输出
     2 info输出并且写入日志
     3 error输出并写入日志
     4 warning输出并写入日志
*/
func Logger(Type int, text string) {
	switch Type {
	case 1:
		fmt.Println("[" + LogTime() + "]" + "[INFO] " + text)
	case 2:
		log := "[" + LogTime() + "]" + "[INFO] " + text
		fmt.Println(log)
		LogFile, err := os.OpenFile("./BPM/Log", os.O_APPEND|os.O_CREATE, fs.ModePerm)
		if err != nil {
			fmt.Println("DEBUG——Logger error,type 2")
		}
		_, err = LogFile.Write([]byte(log + "\n"))
		if err != nil {
			return
		}
		err = LogFile.Close()
		if err != nil {
			return
		}
	case 3:
		log := "[" + LogTime() + "]" + "[ERR] " + text
		fmt.Printf("\033[1;31;40m%s\033[0m\n", log)
		LogFile, err := os.OpenFile("./BPM/Log", os.O_APPEND|os.O_CREATE, fs.ModePerm)
		if err != nil {
			fmt.Println("DEBUG——Logger error,type 3")
		}
		_, err = LogFile.Write([]byte(log + "\n"))
		if err != nil {
			return
		}
		err = LogFile.Close()
		if err != nil {
			return
		}
	case 4:
		log := "[" + LogTime() + "]" + "[WARN] " + text
		fmt.Printf("\033[1;31;40m%s\033[0m\n", log)
		LogFile, err := os.OpenFile("./BPM/Log", os.O_APPEND|os.O_CREATE, fs.ModePerm)
		if err != nil {
			fmt.Println("DEBUG——Logger error,type 4")
		}
		_, err = LogFile.Write([]byte(log + "\n"))
		if err != nil {
			return
		}
		err = LogFile.Close()
		if err != nil {
			return
		}
	}
}

func LogTime() string {
	Year, Mouth, Day := time.Now().Date()
	Hour, Min, Sec := time.Now().Clock()
	return strconv.Itoa(Year) + " " + Mouth.String() + " " + strconv.Itoa(Day) + " " + strconv.Itoa(Hour) + ":" + strconv.Itoa(Min) + ":" + strconv.Itoa(Sec)
}

// DebugFinisher Debug模式下结束操作
func DebugFinisher() error {
	Logger(1, "你没有定义DebugFinisher哦!故没必要开启Debug选项!")
	return nil
}
