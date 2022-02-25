package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"syscall"
)

// SYSLib 加载dll文件

// SYSCmd 加载SysCmd函数
var (
	SYSLib = syscall.NewLazyDLL("SYSCMD.dll")
	SYSCmd = SYSLib.NewProc("SysCmd")
)

var (
	DependDownloadURL DecodeURL
	PluginDownloadURL DecodeURL
	MakeAllDirList    []string
	App               app
)

type app struct {
	TryLink      bool                 `json:"try_link"`
	Debug        bool                 `json:"debug"`
	UserAgent    string               `json:"user_agent"`
	GetPluginURL map[string]DecodeURL `json:"get_plugin_url"`
	GetDependURL map[string]DecodeURL `json:"get_depend_url"`
}

type PluginLog struct {
	File    []string `json:"file"`
	Version string   `json:"version"`
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

type DecodeURL struct {
	URL      string `json:"url"`
	LinkMode string `json:"link_mode"`
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

// urlDecode 对不同LinkMode的DecodeURL进行操作，输出string
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
	} else if url.LinkMode == "0" {
		return url.URL
	} else {
		fmt.Println(errors.New("不支持的LinkMode"))
	}

	return urls
}

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

// IsLastest 检查是否为最新版本
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

// StringToVersion 将string转为version
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

type PackJson struct {
	PackageName string              `json:"Package_name"`
	Version     string              `json:"version"`
	Developer   string              `json:"developer"`
	Depends     map[string][]string `json:"depends"`
	Level       int                 `json:"level"`
	PackageMap  map[string][]string `json:"package_map"`
}
