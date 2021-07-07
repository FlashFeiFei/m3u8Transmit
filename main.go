package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func openFile(file string) (stat os.FileInfo, abs string, err error) {
	open, err := os.Open(file)
	if err != nil {
		return nil, "", errors.New(fmt.Sprint("打开文件失败: ", err))
	}

	stat, err = open.Stat()
	if err != nil {
		return nil, "", errors.New(fmt.Sprint("获取文件信息失败: ", err))
	}

	if stat.IsDir() {
		return nil, "", errors.New("这是一个目录而不是文件")
	}

	abs, err = filepath.Abs(file)
	if err != nil {
		return nil, "", errors.New(fmt.Sprint("获取绝对路径失败:", err))
	}

	return stat, abs, nil
}

//mp4文件转化到ts
//返回ts文件的路径
func mp4ToTs(file string) string {
	stat, abs, err := openFile(file)
	if err != nil {
		log.Println(err)
		return ""
	}

	fileName := "./upload/video/ts/" + strings.Split(stat.Name(), ".")[0] + ".ts"

	//ffmpeg mp4 to ts文件
	arg := []string{
		"-y",
		"-i", abs,
		"-vcodec", "copy",
		"-acodec", "copy",
		"-vbsf", "h264_mp4toannexb",
		fileName,
	}
	command := exec.Command("ffmpeg", arg...)
	output, err := command.CombinedOutput()
	if err != nil {
		log.Println("文件："+abs+"执行mp4转化ts失败", err)
		return ""
	}

	log.Println(string(output))

	newFilePathAbs, _ := filepath.Abs(fileName)
	return newFilePathAbs
}

//ts文件转m3u8
func tsToM3U8(file string) {
	stat, abs, err := openFile(file)
	if err != nil {
		log.Println(err)
		return
	}

	fileName := strings.Split(stat.Name(), ".")[0]

	arg := []string{
		"-y",
		"-i", abs,
		"-hls_time", "15",
		"-hls_segment_filename", "./upload/video/m3u8/" + fileName + "%d.ts",
		"./upload/video/m3u8/" + fileName + ".m3u8",
	}

	command := exec.Command("ffmpeg", arg...)
	output, err := command.CombinedOutput()
	if err != nil {
		log.Println("文件："+abs+"执行ts转化m3u8失败", err)
		return
	}

	log.Println(string(output))

}

//go:embed upload/video
var videoFiles embed.FS

func main() {

	//mp4转ts
	//tsFilePath := mp4ToTs("./upload/video/metadata/banner.mp4")
	//
	////ts转m3u8
	//tsToM3U8(tsFilePath)

	//http服务
	//将请求的url.path中的指定前缀video去掉

	useOS := len(os.Args) > 1 && os.Args[1] == "live"
	tmpfilesHandle := http.StripPrefix("/video/", http.FileServer(getFileSystem(useOS)))
	myAuthFileServerHandle := NewMyAuthFileServerHandle(tmpfilesHandle)

	http.Handle("/video/", myAuthFileServerHandle)
	http.ListenAndServe(":8888", nil)

}

func getFileSystem(useOS bool) http.FileSystem {
	if useOS {
		log.Println("使用 live 模式")
		return http.FS(os.DirFS("upload/video"))
	}

	log.Println("使用内嵌编译模式")

	fsys, err := fs.Sub(videoFiles, "upload/video")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

func NewMyAuthFileServerHandle(handle http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//跨域设置
		w.Header().Set("Access-Control-Allow-Origin", "*")                                                            // 允许访问所有域，可以换成具体url，注意仅具体url才能带cookie信息
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token") //header的类型
		w.Header().Add("Access-Control-Allow-Credentials", "true")                                                    //设置为true，允许ajax异步请求带cookie信息
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")                             //允许请求方法
		w.Header().Set("content-type", "application/json;charset=UTF-8")                                              //返回数据格式是json
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		handle.ServeHTTP(w, r)
	})
}
