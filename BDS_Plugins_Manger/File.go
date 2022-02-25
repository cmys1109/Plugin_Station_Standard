package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

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

// GetDirTreetop 采用递归方式取得目录树树梢
// 目录树树梢顾名思义，是最深处的子目录
func GetDirTreetop(Dir string) ([]string, error) {
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
			_, err := GetDirTreetop(Dir + "\\" + DirList[i])
			if err != nil {
				return nil, err
			}
		}
	} else {
		MakeAllDirList = append(MakeAllDirList, Dir)
	}

	return MakeAllDirList, err
}

// CopyDir 通过DirTreeTop函数读取目录树树梢，并通过os.MkdirAll()快速建立目录树，再将所有文件复制进相应目录
func CopyDir(SrcDir, DestDir string) (error, []string) {
	var DestDirList []string
	MakeAllDirList = nil
	_, err := GetDirTreetop(SrcDir) //读取目录树树梢
	if err != nil {
		fmt.Println(err.Error())
		return err, nil
	}

	for _, paths := range MakeAllDirList { //创建目录树中所有目录
		rel, err := filepath.Rel(SrcDir, paths)
		if err != nil {
			return err, nil
		}
		DestDirList = append(DestDirList, DestDir+"\\"+rel)
		err = os.MkdirAll(DestDir+"\\"+rel, fs.ModePerm)
		if err != nil {
			return err, nil
		}

	}

	var FileLog []string
	err = filepath.Walk(SrcDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			rel, err := filepath.Rel(SrcDir, path)
			if err != nil {
				return err
			}
			err = CopyFile(path, DestDir+"\\"+rel)
			FileLog = append(FileLog, DestDir+"\\"+rel)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err, nil
	}

	return nil, FileLog
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

// GetPathMap 获取指定目录的子目录和所有文件，传回相应的map
func GetPathMap(pathname string) map[string][]string {
	FileList := make(map[string][]string)

	err := filepath.Walk(pathname, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			FileList["File"] = append(FileList["File"], path)
		}
		return nil
	})
	if err != nil {
		Logger(4, "GetPathMap")
		return nil
	}

	FileList["Dir"], err = GetDirTreetop(pathname)
	if err != nil {
		return nil
	}

	return FileList
}
