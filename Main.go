package main

import (
	"./Task"
	"flag"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
)

var addr = flag.String("addr", "172.16.8.21:8283", "http service address")

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	cc := Task.GetConnectBean(c)
	cc.Run()
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html lang="zh-cn">
<head>
    <meta charset="utf-8" />
    <meta http-equiv="X-UA-Compatible" content="IE=edge" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>cola-cloud 构建</title>
    <!-- ZUI 标准版压缩后的 CSS 文件 -->
    <link rel="stylesheet" href="//cdn.bootcss.com/zui/1.9.1/css/zui.min.css" />
    <!-- ZUI Javascript 依赖 jQuery -->
    <script src="//cdn.bootcss.com/zui/1.9.1/lib/jquery/jquery.js"></script>
    <!-- ZUI 标准版压缩后的 JavaScript 文件 -->
    <script src="//cdn.bootcss.com/zui/1.9.1/js/zui.min.js"></script>
    <script>
        window.addEventListener("load", function (evt) {
            var output = document.getElementById("output");
            var ws;
            var print = function (message) {
                var d = document.createElement("div");
                d.innerHTML = message;
                output.appendChild(d);
                output.scrollTop = output.scrollHeight;
            };


            if (ws) {
                mySuccess('你的浏览器不支持ws!')
                return false;
            }
            ws = new WebSocket("{{.}}");
            ws.onopen = function (evt) {
                mySuccess("服务器连接成功！");
            }
            ws.onclose = function (evt) {
                myError("服务器连接被断开！");
                ws = null;
            }
            ws.onmessage = function (evt) {
                console.log('msg:', evt.data);
                if (evt.data) {
                    var data = JSON.parse(evt.data)
                    if (data.out) {
                        print(data.out)
                    }
                    if (data.data) {
                        setSelect(data.data);
                    }
                    if(data.progress!=undefined){
                        if(data.progress ==-1){
                            myError("执行失败！")
                        }
                        if(data.progress == 100){
                            mySuccess("执行成功！")
                        }
                    }
                }
            }
            ws.onerror = function (evt) {
                myError("ERROR: " + evt.data);
            }
            ws.addEventListener('open', function () {
                send("get", "");
            });

            function send(taskName, action) {
                ws.send('{"taskName":"{0}","action":"{1}"}'.format(taskName, action));
                $("#output").html("");

            };

            $("#maven").click(function () {
                send("package", "")
            })
            $("#action").click(function () {
                var value = $("#xjkInput").val();
                send(value,"all")
                // send(value,"push")
            })

            function setSelect(data) {
                var d = data.map(function (e) {
                    return '<option value="' + e.TaskName + '">' + e.TaskName + '</option>'
                })
                $("#xjkInput").html(d);
            };

            function mySuccess(msg) {
                new $.zui.Messager(msg, {
                    type: 'success', // 定义颜色主题
                    time: 500
                }).show();
            };

            function myError(msg) {
                new $.zui.Messager(msg, {
                    type: 'warning', // 定义颜色主题
                    time: 500
                }).show();
            }
        });
    </script>
    <style type="text/css">
        .container{
            position: relative;
            top: 30px;
        }
        #xjkBody {
            position: relative;
            top: 70px;
        }
    </style>
</head>
<body>
<div class="container">
    <div class="row">
<!--        <button id="maven" class="btn btn-primary" type="button">拉取代码并构建</button>-->
        <div class="input-group">
        <div class="input-control search-box search-box-circle has-icon-left has-icon-right search-example" >
            <select id="xjkInput" class="form-control">
            </select>
            <label for="xjkInput" class="input-control-icon-left search-icon" id="maven"><i class="icon icon-play-sign"></i></label>
<!--            <label for="xjkInput" class="input-control-icon-left search-icon">拉取代码并构建</label>-->
        </div>
        <span class="input-group-btn"> <button class="btn btn-primary" id="action" type="button">执行</button> </span>
    </div>
    </div>

    <div id="xjkBody" class=".containe">
        <div>构建结果：</div>
        <hr>
        <pre class="pre-scrollable" id="output">
        </pre>
    </div>
</div>
</body>
</html>
`))
