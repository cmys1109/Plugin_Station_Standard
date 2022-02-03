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

type version struct {
	MajorVersionNumber int
	MinorVersionNumber int
	RevisionNumber     int
}

type Version interface {
	ToString() string
	IsLastest(GetVer version) bool
}

func (version version) ToString() string {
	var versionstr = "V "
	versionstr = versionstr + strconv.Itoa(version.MajorVersionNumber) + "."
	versionstr = versionstr + strconv.Itoa(version.MinorVersionNumber) + "."
	versionstr = versionstr + strconv.Itoa(version.RevisionNumber)
	return versionstr
}

func (version version) IsLastest(GetVer version) bool {
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

func StringToVersion(PluginVer string) (version, error) {
	var PluginSlice = strings.Split(PluginVer, ".")
	if len(PluginSlice) != 3 {
		return version{}, errors.New("version type is not in GNU format")
	}
	minor, err := strconv.Atoi(PluginSlice[1])
	if err != nil {
		return version{}, err
	}
	major, err := strconv.Atoi(strings.Split(PluginSlice[0], " ")[1])
	if err != nil {
		return version{}, err
	}
	re, err := strconv.Atoi(PluginSlice[2])
	if err != nil {
		return version{}, err
	}

	return version{major, minor, re}, nil
}

type app struct {
	PreferredURL string
	AlternateURL string
}

type Details struct {
	Pluginname string     `json:"pluginname"`
	Version    string     `json:"version"`
	Developer  string     `json:"developer"`
	Cmd        [][]string `json:"cmd"`
}

var DownloadURL string

var PluginList map[string]map[string]string

func main() {
	var App app
	var js, _ = ioutil.ReadFile("./config.json")
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

	_, err := os.Stat("./PluginList.json")
	if os.IsNotExist(err) {
		PluginListFile, err := os.OpenFile("./PluginList.json", os.O_RDWR|os.O_CREATE, 0766)
		if err != nil {
			fmt.Println(err.Error())
		}
		_, err = PluginListFile.Write([]byte("{\n  \"pluginkey\": {\n    \"version\": \"V 1.0.0\",\n    \"file_0\": \"./DownloadPage/test.go\"\n  }\n}"))
		if err != nil {
			return
		}
		err = PluginListFile.Close()
		if err != nil {
			return
		}
	}

	fmt.Println("Started!")
	fmt.Println("Enter Plugin's name or enter 0 to exit.")
	var Plugin string
	scanln, err := fmt.Scanln(&Plugin)
	if err != nil {
		scanln = scanln + 1
		return
	}
	if Plugin == "0" {
		fmt.Println("stop")
		return
	}

	if !TryLink(App.PreferredURL) {
		fmt.Println("PreferredURL outdated")
		fmt.Println("Change download address...")
		if !TryLink(App.AlternateURL) {
			fmt.Println("AlternateURL outdated")
			fmt.Println("stop...")
		} else {
			DownloadURL = App.AlternateURL
		}
	} else {
		DownloadURL = App.PreferredURL
	}

	for true {
		time.Sleep(1 * time.Second)
		err := GetPlugin(Plugin)
		if err != nil {
			_ = fmt.Errorf(err.Error())
			return
		}

		Details, err := GetDetails(Plugin)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		err = InstallPlugin(Details, Plugin)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println(Plugin, " Successfully installed!")

		fmt.Print("\n\nInstall other plugin?(y/n)")
		var other string
		scanln, err := fmt.Scanln(&other)
		if err != nil {
			fmt.Println(scanln)
			return
		}

		if other == "n" {
			fmt.Println("stop...")
			break
		}
	}
}

func TryLink(url string) bool {
	fmt.Println("Try to linking...")
	cl := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequest("GET", url+"cmys1109/Plugin-Station/main/"+"README.md", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36")
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

func GetPlugin(PluginKey string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	req, _ := http.NewRequest("GET", DownloadURL+"cmys1109/Plugin-Station/main/Plugins/"+PluginKey, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36")
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
		f, err := os.Create(PluginKey)
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

func GetDetails(PluginKey string) (Details, error) {
	cl := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", DownloadURL+"cmys1109/Plugin-Station/main/Details/"+strings.Split(PluginKey, ".")[0]+".json", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36")
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
		fmt.Println(jsonerr.Error())
		return details, jsonerr
	}
	return details, nil
}

func InstallPlugin(details Details, PluginKey string) error {
	_, err := StringToVersion(details.Version)
	if err != nil {
		return err
	}
	fmt.Println(details.Pluginname, " start to install...")
	fmt.Println("Plugin version:", details.Version)
	fmt.Println("Plugin developer:", details.Developer)
	PluginListStruct := make(map[string]string)
	PluginListStruct["version"] = details.Version
	for Com, Command := range details.Cmd {
		fmt.Println("Run command:", Com)
		switch Command[0] {
		case "unzip":
			err := Unzip(Command[1], Command[2])
			if err != nil {
				return err
			}
		case "copy":
			_, err := copyFile(Command[1], Command[2])
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

	PluginListFile, err := ioutil.ReadFile("./PluginList.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(PluginListFile, &PluginList)
	if err != nil {
		return err
	}
	PluginList[PluginKey] = PluginListStruct
	PluginListByte, err := json.Marshal(PluginList)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("./PluginList.json", PluginListByte, fs.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func UpdatePlugin(PluginKey string, NowVer version) error {
	details, err := GetDetails(PluginKey)
	if err != nil {
		return err
	}
	NowVersion, err := StringToVersion(details.Version)
	if err != nil {
		return err
	}

	if NowVer.IsLastest(NowVersion) {
		return nil
	}
	err = GetPlugin(PluginKey)
	if err != nil {
		return err
	}
	err = InstallPlugin(details, PluginKey)
	if err != nil {
		return err
	}

	return nil
}

//func UninstallPlugin(PluginKey string)

func Unzip(zipFile string, destDir string) error { //https://www.jianshu.com/p/4593cfffb9e9
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	//defer func(zipReader *zip.ReadCloser) {
	//	err := zipReader.Close()
	//	if err != nil {
	//
	//	}
	//}(zipReader)

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
		err := zipReader.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFile(srcFile, destFile string) (int64, error) {
	file1, err := os.Open(srcFile)
	if err != nil {
		return 0, err
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
				return 0, err
			}
		} else {
			return 0, err
		}
	}
	file2, err = os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return 0, nil
	}
	defer func(file1 *os.File) {
		err := file1.Close()
		if err != nil {

		}
	}(file1)
	defer func(file2 *os.File) {
		err := file2.Close()
		if err != nil {

		}
	}(file2)

	return io.Copy(file2, file1)
}

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
