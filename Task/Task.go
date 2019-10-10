package Task

import (
	"bufio"
	"fmt"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Charset string

const (
	UTF8    = Charset("UTF-8")
	GB18030 = Charset("GB18030")
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Msg struct {
	TaskName string `json:"taskName"`
	Action   string `json:"action"`
}

type ResultMsg struct {
	Msg      string      `json:"msg"`
	Progress int         `json:"progress"`
	Data     interface{} `json:"data"`
	Out      string      `json:"out"`
}

type Action struct {
	ActionName string
	ActionCmd  string
}

type Task struct {
	TaskName string
	Actions  *[]Action
}

type Connect struct {
	ConId       string
	Con         *websocket.Conn
	BashPath    string `工作目录`
	ActionInfo  *[]Task
	BuildString string
}

func GetConnectBean(con *websocket.Conn) Connect {
	return Connect{Con: con}
}

type ProjectTree struct {
	ProjectName string
	ProjectPath string
}

var DefaultBashPath = "D:\\code\\sandbox\\cola-cloud"

//var DefaultBuildString = "git pull&&mvn clean"
var DefaultBuildString = "git pull&&mvn clean&&mvn -T 1C package -DskipTests -Dmaven.compile.fork=true"

func (connect Connect) Run() {
	defer connect.Con.Close()
	if connect.BashPath == "" {
		connect.BashPath = DefaultBashPath
	}
	if connect.BuildString == "" {
		connect.BuildString = DefaultBuildString
	}
	if connect.ActionInfo == nil {
		err := filepath.Walk(connect.BashPath, ListFunc)
		if err != nil {
			connect.Write(ResultMsg{Out: "filepath.Walk() returned %v\n"}.GetByte())
		}
		if len(listFile) != 0 {
			var ts []Task
			for _, v := range listFile {
				t := Task{TaskName: v.ProjectName}
				a1 := Action{ActionName: "all", ActionCmd: fmt.Sprintf("docker build -t goodrain.me/%s %s", v.ProjectName, v.ProjectPath) + "&&" + fmt.Sprintf("docker push goodrain.me/%s", v.ProjectName)}
				//a2 := Action{ActionName: "push", ActionCmd: fmt.Sprintf("docker push goodrain.me/%s", v.ProjectName)}
				t.Actions = &[]Action{a1}
				ts = append(ts, t)
			}
			connect.ActionInfo = &ts
		}
	}
	for {
		mt, message, err := connect.Con.ReadMessage()
		if err != nil {
			break
		}
		if mt == websocket.TextMessage {

			var data Msg
			if err := json.Unmarshal(message, &data); err != nil {
				fmt.Println(string(message))
				log.Println("解析数据出错")
				connect.Write(ResultMsg{Msg: "解析数据出错", Progress: -1}.GetByte())
			} else {
				b := xjkSelect(connect, data)
				if !b {
					connect.Write(ResultMsg{Msg: "不能理解", Progress: -1}.GetByte())
				}
			}
		} else {
			err = connect.Con.WriteMessage(mt, message)
			if err != nil {
				log.Println("write:", err)
				break
			}
		}
	}
}

func (connect Connect) maven() {
	connect.execCommand(connect.BuildString)
}

var listFile []ProjectTree //获取文件列表

func xjkSelect(connect Connect, data Msg) bool {
	if strings.Compare("package", data.TaskName) == 0 {
		connect.maven()
		return true
	}
	if strings.Compare("get", data.TaskName) == 0 {
		connect.Write(ResultMsg{Msg: "ok", Progress: 50, Data: &connect.ActionInfo}.GetByte())
		return true
	}
	for _, task := range *connect.ActionInfo {
		if strings.Compare(task.TaskName, data.TaskName) == 0 {
			for _, action := range *task.Actions {
				if strings.Compare(action.ActionName, data.Action) == 0 {
					connect.execCommand(action.ActionCmd)
					return true
				}
			}
		}
	}
	return false
}

func (connect Connect) Write(b []byte) {
	connect.Con.WriteMessage(websocket.TextMessage, b)
}

func (connect Connect) Println(str []string) {
	msg := ResultMsg{Out: strings.Join(str, " ")}
	connect.Write(msg.GetByte())
}

func (connect Connect) execCommand(command string) bool {
	var flag bool = true
	for _, v := range strings.Split(command, "&&") {
		execCommand := connect.ExecCommand(Parse(strings.Trim(v, " ")))
		if !execCommand {
			flag = false
		}
	}

	if flag {
		connect.Write(ResultMsg{Progress: 100, Msg: "执行成功！"}.GetByte())
	} else {
		connect.Write(ResultMsg{Progress: -1, Msg: "执行失败！"}.GetByte())
	}
	return true
}

func (connect Connect) ExecCommand(commandName string, params []string) bool {
	cmd := exec.Command(commandName, params...)
	cmd.Dir = connect.BashPath

	//显示运行的命令
	connect.Println(cmd.Args)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		connect.Println([]string{err.Error()})
		return false
	}
	cmd.Start()
	reader := bufio.NewReader(stdout)

	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadBytes('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		var str string
		if osType == "windows" {
			str = ConvertByte2String(line, "GB18030")
		} else {
			str = ConvertByte2String(line, "UTF-8")
		}

		connect.Write(ResultMsg{Out: str}.GetByte())
	}

	cmd.Wait()
	return true
}

func (msg ResultMsg) GetByte() []byte {
	if by, err := jsoniter.Marshal(msg); err == nil {
		return by
	}
	var bb []byte
	return bb
}

func Parse(str string) (s string, string []string) {
	split := strings.Split(str, " ")
	return split[0], split[1:]
}

var (
	osType = os.Getenv("GOOS") // 获取系统类型
)

func ConvertByte2String(byte []byte, charset Charset) string {
	var str string
	switch charset {
	case GB18030:
		var decodeBytes, _ = simplifiedchinese.GB18030.NewDecoder().Bytes(byte)
		str = string(decodeBytes)
	case UTF8:
		fallthrough
	default:
		str = string(byte)
	}
	return str
}

func ListFunc(path string, f os.FileInfo, err error) error {
	var strRet string
	//strRet, _ = os.Getwd()
	p := ProjectTree{}

	if osType == "windows" {
		strRet += "\\"
	} else if osType == "linux" {
		strRet += "/"
	}

	if f == nil {
		return err
	}
	if f.IsDir() {
		return nil
	}

	strRet += path //+ "\r\n"
	//用strings.HasSuffix(src, suffix)//判断src中是否包含 suffix结尾
	ok := strings.HasSuffix(strRet, "dockerRun.txt")
	if ok {
		f, err := os.Open(strRet)
		defer f.Close()
		bfRd := bufio.NewReader(f)
		if err == nil {
			line, _ := bfRd.ReadBytes('\n')
			str := string(line)
			if strings.HasSuffix(str, "\n") {
				str = str[0 : len(str)-1]
			}
			if strings.HasSuffix(str, "\r") {
				str = str[0 : len(str)-1]
			}
			p.ProjectName = str
		}

		p.ProjectPath = strRet[0 : len(strRet)-14]
		listFile = append(listFile, p) //将目录push到listfile []string中
	}
	return nil
}
