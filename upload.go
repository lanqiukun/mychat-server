package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	maxUploadSize = 20 << 20
	ImageType     = "image"
	VideoType     = "video"
	AudioType     = "audio"
)

type File struct {
	//
	Type uint   `json:"type"`
	Url  string `json:"url"`
}

type UploadResponse struct {
	//成功0
	//未知错误1
	//已知错误2
	Status uint8  `json:"status"`
	Files  []File `json:"files"`
	Reason string `json:"reason,omitempty"`
}

func uploadFile(w http.ResponseWriter, r *http.Request) {

	if isPreflight := cors(&w, r); isPreflight == true {
		return
	}
	//先验证凭据

	var uploadResponse UploadResponse
	uploadResponse.Status = 1

	defer func() {

		if uploadResponseJson, err := json.Marshal(uploadResponse); err == nil {
			w.Write(uploadResponseJson)
		} else {
			println(err.Error())
		}

	}()

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize) // 20 Mb
	err := r.ParseMultipartForm(maxUploadSize)             // grab the multipart form
	if err != nil {
		fmt.Fprintln(w, err)
		uploadResponse.Status = 2
		uploadResponse.Reason = err.Error()
		return
	}

	//get the *fileheaders
	files := r.MultipartForm.File["multiplefiles"] // grab the filenames

	for _, file := range files { // loop through the files one by one
		f, err := file.Open()
		defer f.Close()
		if err != nil {
			println(err.Error())
			uploadResponse.Status = 2
			uploadResponse.Reason = err.Error()
			return
		}

		// //检测文件类型////////////////////////////////
		buffer := make([]byte, 512)
		_, err = f.Read(buffer)
		if err != nil {
			println(err.Error())
			uploadResponse.Status = 2
			uploadResponse.Reason = err.Error()
			return
		}
		fileType := http.DetectContentType(buffer)

		fileName := strconv.FormatUint(rand.Uint64(), 10) + "."

		var clientFileInfo File

		var dir string = "upload/"
		if strings.Index(fileType, ImageType) != -1 {
			dir += ImageType
			fileName += fileType[len(ImageType)+1:]
			clientFileInfo.Type = 2
		} else if strings.Index(fileType, VideoType) != -1 {
			dir += VideoType
			fileName += fileType[len(VideoType)+1:]
			clientFileInfo.Type = 3
		} else if strings.Index(fileType, AudioType) != -1 {
			dir += AudioType
			fileName += fileType[len(AudioType)+1:]
			clientFileInfo.Type = 4
		} else {
			dir += "file"
			fileName = file.Filename
			clientFileInfo.Type = 5
		}
		//////////////////////////////////////////////

		// set position back to start.
		if _, err := f.Seek(0, 0); err != nil {
			println(err.Error())
			uploadResponse.Status = 2
			uploadResponse.Reason = err.Error()
			return
		}

		result := dir + "/" + fileName
		clientFileInfo.Url = result

		out, err := os.Create(result)
		defer out.Close()
		if err != nil {
			println(err.Error())
			uploadResponse.Status = 2
			uploadResponse.Reason = err.Error()
			return
		}

		_, err = io.Copy(out, f)
		if err != nil {
			println(err.Error())
			uploadResponse.Status = 2
			uploadResponse.Reason = err.Error()
			return
		}

		uploadResponse.Files = append(uploadResponse.Files, clientFileInfo)

	}
	uploadResponse.Status = 0
}
