
# BDS_Plugin_Manager  

---
## About
我个人给其定义是BDS服务端的一个插件包管理工具，我对包管理的了解尚浅，所以此程序的一些逻辑也不够成熟，是以最直观化的思维去解决的问题的  

这是一个BDS插件管理工具，正在逐步开发中  
开源于[客户端]文件夹中

关于更详细的一些内容，请查看main.go注释，做出了一些解释

---

### 操作指令：
```
命令：install   + 空格 + [PluginKey]——————安装插件  
命令：uninstall + 空格 + [PluginKey]——————卸载插件  
命令：update    + 空格 + [PluginKey]——————更新插件  
命令："update -a"—————————————————————更新所有插件  
命令："0"————————————————————————————————退出程序
命令：depend    + 空格 + [Depend]—————————安装依赖
命令：undepend  + 空格 + [Depend]—————————卸载依赖
命令："list"—————————————列出已安装的插件和依赖列表
```

---
# Plugin_Station  

使用GitHub仓库作为插件储存库  
插件作者可以通过按照规范在[Plugin_Station](https://github.com/cmys1109/Plugin-Station) 申请PR来提交插件  
或者申请成为仓库协作者  
提交插件以及插件详情json，客户端程序会按照作者提交的json内容自动安装插件

插件下载API插件包标准  v220212
------
##  仓库文件存放结构
Plugin文件只能为单个，如果是多个文件可以达成zip压缩包上传。仅限zip，因为客户端解压压缩包方案仅支持zip  

确保Plugin文件名除去后缀后和details.json文件名去除后缀后相同  

例子：插件文件名[123.zip]，详情包名[123.json]  
PluginKey为[123.zip]

Plugin
 ###  --/Plugins/
 ####  --PluginFile  
 ###  --/Details/
 ####  --PluginName.json

---

## Detail.json

```
"pluginname"  :"  "  string
"version"     :"  "  string
"developer"   :"  "  string        
"depends"     :{                  |struct {//结构定义
                "depends":[],     |        depends []string
                "plugins":[]      |        plugins []string 
               }                  |      }  
"level"       : 2    int
"install_cmd" :[  ]  []string
"update_cmd"  :[  ]  []string
```

## [example](https://github.com/cmys1109/Plugin-Station/blob/main/Details/123.json)  


------
##  cmd规范

cmd为一个二维数组，元素值为string

客户端在拿到detail.json的数据后会解析cmd  
并且按顺序进行操作  
### 关于编写cmd的提醒：
```
1.在编写Install_cmd和Update_cmd时请注意
  下载的文件本体存放于./temp目录中
  
2.注意点：数组中的key请按操作顺序排序！
```
---
### cmd所提供的方法：  

1. 解压，数组内容：[``"unzip"``,``<压缩包路径>``,``<解压至路径>``]  
将<压缩包路径>的zip包解压至<解压至路径>(不存在会自行创建)  
2. 复制，数组内容：[``"copy"``,``<文件路径>``,``<复制至路径>``,``<复制后命名为>``]
3. 删除，数组内容：[``"del"``,``<文件或目录名>``]  
4. 运行系统命令，数组内容:[``"syscmd“``，``<系统命令>``]
### 关于cmd方法的特别提醒：
```
将通过统一的函数来进行运行cmd
其中常规的操作，如：解压、复制、删除都会记录在PluginList.json中
删除操作的目标文件必须在PluginList.json的记录中，否则无法进行删除操作

但是syscmd方法由于其特殊性不提供记录，所以没有任何限制
但是正式由于其特殊性，并且其稳定性尚未验证，故请谨慎使用
如无必要请不要使用此方法
```


