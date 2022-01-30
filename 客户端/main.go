package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type app struct {
	PreferredURL string
	AlternateURL string
}

type Details struct {
	Pluginname string              `json:"pluginname"`
	Version    string              `json:"version"`
	Developer  string              `json:"developer"`
	Cmd        map[string][]string `json:"cmd"`
}

var DownloadURL string

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

	if TryLink(App.PreferredURL) == false {
		fmt.Println("PreferredURL outdated")
		fmt.Println("Change download address...")
		if TryLink(App.AlternateURL) == false {
			fmt.Println("AlternateURL outdated")
			fmt.Println("stop...")
		} else {
			DownloadURL = App.AlternateURL
		}
	} else {
		DownloadURL = App.PreferredURL
	}

	for true {

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

		err = InstallPlugin(Details)
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

func GetPlugin(Plugin string) error {
	time.Sleep(1 * time.Second)
	client := &http.Client{Timeout: 60 * time.Second}
	req, _ := http.NewRequest("GET", DownloadURL+"cmys1109/Plugin-Station/main/Plugins/"+Plugin, nil)
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
		f, err := os.Create(Plugin)
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

func GetDetails(Plugin string) (Details, error) {
	cl := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", DownloadURL+"cmys1109/Plugin-Station/main/Details/"+strings.Split(Plugin, ".")[0]+".json", nil)
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

func InstallPlugin(details Details) error {
	fmt.Println(details.Pluginname, " start to install...")
	fmt.Println("Plugin version:", details.Version)
	fmt.Println("Plugin developer:", details.Developer)
	for CommandName, Command := range details.Cmd {
		fmt.Println("Run command:", CommandName)
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
		case "del":
			err := DelFileOrDir(Command[1])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

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
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(fpath, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
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

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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
		return 0, err
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
