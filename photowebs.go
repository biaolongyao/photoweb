package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

const (
	ListDir      = 0x0001
	UPLOAD_DIRS  = "./uploads"
	TEMPLATE_DIR = "./views"
)

var templates = make(map[string]*template.Template)

//将页面模板加载。
func init() {

	fileInfoArr, err := ioutil.ReadDir(TEMPLATE_DIR)
	if err != nil {
		panic(err)
		return
	}
	var templateName, templatePath string
	for _, fileInfo := range fileInfoArr {
		templateName = fileInfo.Name()
		if ext := path.Ext(templateName); ext != ".html" {
			continue
		}
		templatePath = TEMPLATE_DIR + "/" + templateName
		fmt.Println(templatePath)
		log.Println("loading template:", templatePath)
		t := template.Must(template.ParseFiles(templatePath))
		fmt.Println(templateName)
		templates[templateName] = t
	}
}

//统一捕获服务端的内部错误，panic的运行会是程序停止运行，safeHandler来解决这一问题。
func check(err error) {
	if err != nil {
		panic(err)
	}
}

func renderHtml(w http.ResponseWriter, tmpl string, locals map[string]interface{}) {
	err := templates[tmpl].Execute(w, locals)
	check(err)
}

func isExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

//这是上传图片的handler
func uploadHandlers(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderHtml(w, "upload.html", nil)
	}
	if r.Method == "POST" {
		//r.FormFile用来获取接受的请求的文件句柄。
		f, h, err := r.FormFile("image")
		check(err)
		filename := h.Filename
		fmt.Println(filename)
		defer f.Close()
		//t,err :=ioutil.TempFile(UPLOAD_DIRS,filename)
		t, err := os.Create(UPLOAD_DIRS + "/" + filename)
		check(err)
		defer t.Close()
		_, err = io.Copy(t, f)
		check(err)
		http.Redirect(w, r, "/view?id="+filename, http.StatusFound)

	}
}

func viewHandlers(w http.ResponseWriter, r *http.Request) {
	imageId := r.FormValue("id")
	imagePath := UPLOAD_DIRS + "/" + imageId
	fmt.Println("---------------------")
	fmt.Println(imagePath)
	fmt.Println("---------------------")
	if exists := isExists(imagePath); !exists {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Tpye", "image")
	//读取图片，并将图片写入到http.ResponseWriter
	http.ServeFile(w, r, imagePath)
}

func listHandlers(w http.ResponseWriter, r *http.Request) {
	//遍历指定目录，他读取目录并返回排好序的文件以及子目录名
	FileInfoArr, err := ioutil.ReadDir("./uploads")
	check(err)
	locals := make(map[string]interface{})
	images := []string{}
	for _, fileInfo := range FileInfoArr {
		images = append(images, fileInfo.Name())
	}
	locals["images"] = images
	renderHtml(w, "list.html", locals)
}

//这个函数，是传入一个业务函数，并且调用这个业务逻辑函数，若此业务逻辑函数引发了panic，
// defer会通过recover捕获这个异常，然后正常处理，使后续程序正常运行。
func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered panic: %s\n", r)
			}
		}()
		fn(w, r)
	}
}

func staticDirHandler(mux *http.ServeMux, prefix string, staticDir string, flags int) {
	mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		file := staticDir + r.URL.Path[len(prefix)-1:]
		if (flags & ListDir) == 0 {
			if exists := isExists(file); !exists {
				http.NotFound(w, r)
				return
			}
		}
		http.ServeFile(w, r, file)
	})
}

func main() {
	mux := http.NewServeMux()
	staticDirHandler(mux, "/assets/", "./public", 0)
	mux.HandleFunc("/", safeHandler(listHandlers))
	mux.HandleFunc("/view", safeHandler(viewHandlers))
	mux.HandleFunc("/upload", safeHandler(uploadHandlers))
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal("ListenAndServe:", err.Error())
	}
}
