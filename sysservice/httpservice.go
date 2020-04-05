package sysservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/duanhf2012/origin/sysmodule"

	"github.com/duanhf2012/origin/rpc"

	"github.com/duanhf2012/origin/cluster"
	"github.com/duanhf2012/origin/network"
	"github.com/duanhf2012/origin/service"
	"github.com/duanhf2012/origin/util/uuid"
)

//http redirect
type HttpRedirectData struct {
	Url string
	CookieList []*http.Cookie
}

type HttpRequest struct {
	URL      string
	Header http.Header
	Body   string

	paramStr string //http://127.0.0.1:7001/aaa/bbb?aa=1 paramStr is:aa=1
	mapParam map[string]string
}

type HttpRespone struct {
	Respone      []byte
	RedirectData HttpRedirectData
}

type HTTP_METHOD int

const (
	METHOD_NONE HTTP_METHOD = iota
	METHOD_GET
	METHOD_POST

	METHOD_INVALID
	//METHOD_PUT  HTTP_METHOD = 2
)


type ControllerMapsType map[string]reflect.Value
type RouterType int
const (
	FuncCall RouterType = iota+1
	StaticResource
)

type routerMatchData struct {
	callpath   string
	matchURL   string
	routerType int8 //0表示函数调用  1表示静态资源
}

type serveHTTPRouterMux struct {
	httpFiltrateList [] HttpFiltrate
	allowOrigin      bool
}

type HttpService struct {
	service.Service

	httpServer network.HttpServer
	controllerMaps ControllerMapsType
	serverHTTPMux  serveHTTPRouterMux

	postAliasUrl map[HTTP_METHOD] map[string]routerMatchData //url地址，对应本service地址
	staticRouterResource map[HTTP_METHOD] routerStaticResoutceData

	httpProcessor IHttpProcessor
}

type routerStaticResoutceData struct {
	localPath string
	method    HTTP_METHOD
}


type HttpHandle func(request *HttpRequest, resp *HttpRespone) error

func AnalysisRouterUrl(url string) (string, error) {

	//替换所有空格
	url = strings.ReplaceAll(url, " ", "")
	if len(url) <= 1 || url[0] != '/' {
		return "", fmt.Errorf("url %s format is error!", url)
	}

	//去掉尾部的/
	return strings.Trim(url, "/"), nil
}

func (slf *HttpRequest) Query(key string) (string, bool) {
	if slf.mapParam == nil {
		slf.mapParam = make(map[string]string)
		//分析字符串
		slf.paramStr = strings.Trim(slf.paramStr, "/")
		paramStrList := strings.Split(slf.paramStr, "&")
		for _, val := range paramStrList {
			index := strings.Index(val, "=")
			if index >= 0 {
				slf.mapParam[val[0:index]] = val[index+1:]
			}
		}
	}

	ret, ok := slf.mapParam[key]
	return ret, ok
}

type HttpSession struct {
	request *HttpRequest
	resp *HttpRespone
}

type HttpProcessor struct {
	httpSessionChan chan *HttpSession
	pathRouter map[HTTP_METHOD] map[string] routerMatchData //url地址，对应本service地址
	staticRouterResource map[HTTP_METHOD] routerStaticResoutceData
}

var Default_HttpSessionChannelNum = 100000

func NewHttpProcessor() *HttpProcessor{
	httpProcessor := &HttpProcessor{}
	httpProcessor.httpSessionChan =make(chan *HttpSession,Default_HttpSessionChannelNum)
	httpProcessor.staticRouterResource = map[HTTP_METHOD] routerStaticResoutceData{}
	httpProcessor.pathRouter =map[HTTP_METHOD] map[string] routerMatchData{}

	for i:=METHOD_NONE+1;i<METHOD_INVALID;i++{
		httpProcessor.pathRouter[i] = map[string] routerMatchData{}
	}

	return httpProcessor
}

var Default_HttpProcessor *HttpProcessor= NewHttpProcessor()

type IHttpProcessor interface {
	PutHttpSession(httpSession *HttpSession)
	GetHttpSessionChannel() chan *HttpSession
	RegRouter(method HTTP_METHOD, url string, handle HttpHandle)
	Router(session *HttpSession)
}

func (slf *HttpProcessor) PutHttpSession(httpSession *HttpSession){
	slf.httpSessionChan <- httpSession
}

func (slf *HttpProcessor) GetHttpSessionChannel()  chan *HttpSession {
	return slf.httpSessionChan
}

func (slf *HttpProcessor) RegRouter(method HTTP_METHOD, url string, handle HttpHandle){
	pathRouter
}

func (slf *HttpProcessor) Router(session *HttpSession){

}

func (slf *HttpService) SetHttpProcessor(httpProcessor IHttpProcessor) {
	slf.httpProcessor = httpProcessor
}

func (slf *HttpService) Request(method HTTP_METHOD, url string, handle HttpHandle) error {
	fnpath := runtime.FuncForPC(reflect.ValueOf(handle).Pointer()).Name()

	sidx := strings.LastIndex(fnpath, "*")
	if sidx == -1 {
		return errors.New(fmt.Sprintf("http post func path is error, %s\n", fnpath))
	}

	eidx := strings.LastIndex(fnpath, "-fm")
	if sidx == -1 {
		return errors.New(fmt.Sprintf("http post func path is error, %s\n", fnpath))
	}
	callpath := fnpath[sidx+1 : eidx]
	ridx := strings.LastIndex(callpath, ")")
	if ridx == -1 {
		return errors.New(fmt.Sprintf("http post func path is error, %s\n", fnpath))
	}

	hidx := strings.LastIndex(callpath, "HTTP_")
	if hidx == -1 {
		return errors.New(fmt.Sprintf("http post func not contain HTTP_, %s\n", fnpath))
	}

	callpath = strings.ReplaceAll(callpath, ")", "")

	var r RouterMatchData
	var matchURL string
	var err error
	r.routerType = 0
	r.callpath = "_" + callpath
	matchURL, err = AnalysisRouterUrl(url)
	if err != nil {
		return err
	}

	var strMethod string
	if method == METHOD_GET {
		strMethod = "GET"
	} else if method == METHOD_POST {
		strMethod = "POST"
	} else {
		return nil
	}

	postAliasUrl[strMethod][matchURL] = r

	return nil
}

func Post(url string, handle HttpHandle) error {
	return Request(METHOD_POST, url, handle)
}

func Get(url string, handle HttpHandle) error {
	return Request(METHOD_GET, url, handle)
}

func (slf *HttpService) OnInit() error {
	slf.serverHTTPMux = ServeHTTPRouterMux{}
	slf.httpserver.Init(slf.port, &slf.serverHTTPMux, 10*time.Second, 10*time.Second)
	if slf.ishttps == true {
		slf.httpserver.SetHttps(slf.certfile, slf.keyfile)
	}
	return nil
}

func (slf *HttpService) SetAlowOrigin(allowOrigin bool) {
	slf.allowOrigin = allowOrigin
}

func (slf *HttpService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if slf.allowOrigin == true {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers",
				"Action, Module") //有使用自定义头 需要这个,Action, Module是例子
		}
	}

	if r.Method == "OPTIONS" {
		return
	}

	methodRouter, bok := postAliasUrl[r.Method]
	if bok == false {
		writeRespone(w, http.StatusNotFound, fmt.Sprint("Can not support method."))
		return
	}

	//权限验证
	var errRet error
	for _, filter := range slf.httpfiltrateList {
		ret := filter(r.URL.Path, w, r)
		if ret == nil {
			errRet = nil
			break
		} else {
			errRet = ret
		}
	}
	if errRet != nil {
		writeRespone(w, http.StatusOK, errRet.Error())
		return
	}

	url := strings.Trim(r.URL.Path, "/")
	var strCallPath string
	matchData, ok := methodRouter[url]
	if ok == true {
		strCallPath = matchData.callpath
	} else {
		//如果是资源处理
		for k, v := range staticRouterResource {
			idx := strings.Index(url, k)
			if idx != -1 {
				staticServer(k, v, w, r)
				return
			}
		}

		// 拼接得到rpc服务的名称
		vstr := strings.Split(url, "/")
		if len(vstr) < 2 {
			writeRespone(w, http.StatusNotFound, "Cannot find path.")
			return
		}
		strCallPath = "_" + vstr[0] + ".HTTP_" + vstr[1]
	}

	defer r.Body.Close()
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeRespone(w, http.StatusBadRequest, "")
		return
	}

	request := HttpRequest{r.Header, string(msg), r.URL.RawQuery, nil, r.URL.Path}
	var resp HttpRespone
	//resp.Resp = w
	timeFuncStart := time.Now()
	err = cluster.InstanceClusterMgr().Call(strCallPath, &request, &resp)

	timeFuncPass := time.Since(timeFuncStart)
	if bPrintRequestTime {
		service.GetLogger().Printf(service.LEVER_INFO, "HttpServer Time : %s url : %s\n", timeFuncPass, strCallPath)
	}
	if err != nil {
		writeRespone(w, http.StatusBadRequest, fmt.Sprint(err))
	} else {
		if resp.RedirectData.Url != "" {
			resp.redirects(&w, r)
		} else {
			writeRespone(w, http.StatusOK, string(resp.Respone))
		}

	}
}

// CkResourceDir 检查静态资源文件夹路径
func SetStaticResource(method HTTP_METHOD, urlpath string, dirname string) error {
	_, err := os.Stat(dirname)
	if err != nil {
		return err
	}
	matchURL, berr := AnalysisRouterUrl(urlpath)
	if berr != nil {
		return berr
	}

	var routerData RouterStaticResoutceData
	if method == METHOD_GET {
		routerData.method = "GET"
	} else if method == METHOD_POST {
		routerData.method = "POST"
	} else {
		return nil
	}
	routerData.localpath = dirname

	staticRouterResource[matchURL] = routerData
	return nil
}

func writeRespone(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

type HttpFiltrate func(path string, w http.ResponseWriter, r *http.Request) error

func (slf *HttpService) AppendHttpFiltrate(fun HttpFiltrate) bool {
	slf.serverHTTPMux.httpfiltrateList = append(slf.serverHTTPMux.httpfiltrateList, fun)

	return false
}

func (slf *HttpService) OnRun() bool {

	slf.httpserver.Start()
	return false
}

func NewHttpServerService(port uint16) *HttpServerService {
	http := new(HttpServerService)

	http.port = port
	return http
}

func (slf *HttpService) OnDestory() error {
	return nil
}

func (slf *HttpService) OnSetupService(iservice service.IService) {
	rpc.RegisterName(iservice.GetServiceName(), "HTTP_", iservice)
}

func (slf *HttpServerService) OnRemoveService(iservice service.IService) {
	return
}

func (slf *HttpService) SetPrintRequestTime(isPrint bool) {
	bPrintRequestTime = isPrint
}

func staticServer(routerUrl string, routerData RouterStaticResoutceData, w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	idx := strings.Index(upath, routerUrl)
	subPath := strings.Trim(upath[idx+len(routerUrl):], "/")

	destLocalPath := routerData.localpath + subPath

	writeResp := func(status int, msg string) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(status)
		w.Write([]byte(msg))
	}

	switch r.Method {
	//获取资源
	case "GET":
		//判断文件夹是否存在
		_, err := os.Stat(destLocalPath)
		if err == nil {
			http.ServeFile(w, r, destLocalPath)
		} else {
			writeResp(http.StatusNotFound, "")
			return
		}
	//上传资源
	case "POST":
		// 在这儿处理例外路由接口
		/*
			var errRet error
			for _, filter := range slf.httpfiltrateList {
				ret := filter(r.URL.Path, w, r)
				if ret == nil {
					errRet = nil
					break
				} else {
					errRet = ret
				}
			}
			if errRet != nil {
				w.Write([]byte(errRet.Error()))
				return
			}*/
		r.ParseMultipartForm(32 << 20) // max memory is set to 32MB
		resourceFile, resourceFileHeader, err := r.FormFile("file")
		if err != nil {
			fmt.Println(err)
			writeResp(http.StatusNotFound, err.Error())
			return
		}
		defer resourceFile.Close()
		//重新拼接文件名
		imgFormat := strings.Split(resourceFileHeader.Filename, ".")
		if len(imgFormat) < 2 {
			writeResp(http.StatusNotFound, "not a file")
			return
		}
		filePrefixName := uuid.Rand().HexEx()
		fileName := filePrefixName + "." + imgFormat[len(imgFormat)-1]
		//创建文件
		localpath := fmt.Sprintf("%s%s", destLocalPath, fileName)
		localfd, err := os.OpenFile(localpath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(err)
			writeResp(http.StatusNotFound, "upload fail")
			return
		}
		defer localfd.Close()
		io.Copy(localfd, resourceFile)
		writeResp(http.StatusOK, upath+"/"+fileName)
	}

}

func (slf *HttpService) GetMethod(strCallPath string) (*reflect.Value, error) {
	value, ok := slf.controllerMaps[strCallPath]
	if ok == false {
		err := fmt.Errorf("not find api")
		return nil, err
	}

	return &value, nil
}

func (slf *HttpService) SetHttps(certfile string, keyfile string) bool {
	if certfile == "" || keyfile == "" {
		return false
	}

	slf.ishttps = true
	slf.certfile = certfile
	slf.keyfile = keyfile

	return true
}

//序列化后写入Respone
func (slf *HttpRespone) WriteRespne(v interface{}) error {
	StrRet, retErr := json.Marshal(v)
	if retErr != nil {
		slf.Respone = []byte(`{"Code": 2,"Message":"service error"}`)
		service.GetLogger().Printf(sysmodule.LEVER_ERROR, "Json Marshal Error:%v\n", retErr)
	} else {
		slf.Respone = StrRet
	}

	return retErr
}

func (slf *HttpRespone) WriteRespones(Code int32, Msg string, Data interface{}) {

	var StrRet string
	//判断是否有错误码
	if Code > 0 {
		StrRet = fmt.Sprintf(`{"RCode": %d,"RMsg":"%s"}`, Code, Msg)
	} else {
		if Data == nil {
			if Msg != "" {
				StrRet = fmt.Sprintf(`{"RCode": 0,"RMsg":"%s"}`, Msg)
			} else {
				StrRet = `{"RCode": 0}`
			}
		} else {
			if reflect.TypeOf(Data).Kind() == reflect.String {
				StrRet = fmt.Sprintf(`{"RCode": %d , "Data": "%s"}`, Code, Data)
			} else {
				JsonRet, Err := json.Marshal(Data)
				if Err != nil {
					service.GetLogger().Printf(sysmodule.LEVER_ERROR, "common WriteRespone Json Marshal Err %+v", Data)
				} else {
					StrRet = fmt.Sprintf(`{"RCode": %d , "Data": %s}`, Code, JsonRet)
				}
			}
		}
	}
	slf.Respone = []byte(StrRet)
}

func (slf *HttpRespone) Redirect(url string, cookieList []*http.Cookie) {
	slf.RedirectData.Url = url
	slf.RedirectData.CookieList = cookieList
}

func (slf *HttpRespone) redirects(w *http.ResponseWriter, req *http.Request) {
	if slf.RedirectData.CookieList != nil {
		for _, v := range slf.RedirectData.CookieList {
			http.SetCookie(*w, v)
		}
	}

	http.Redirect(*w, req, slf.RedirectData.Url,
		// see @andreiavrammsd comment: often 307 > 301
		http.StatusTemporaryRedirect)
}