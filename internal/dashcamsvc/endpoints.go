package dashcamsvc

import (
	"net/http"
	"os"
	"io/ioutil"
	"path"
)

func StartRecording(dcsvc *DashCamService) http.HandlerFunc{
	return func(wr http.ResponseWriter, req *http.Request) {
		go dcsvc.StartRecording()
	}
}

func StopRecording(dcsvc *DashCamService) http.HandlerFunc{
	return func(wr http.ResponseWriter, req *http.Request) {
		go dcsvc.StopRecording()
	}
}

func GetStill(dcsvc *DashCamService) http.HandlerFunc{
	return func(wr http.ResponseWriter, req *http.Request) {
		tmpDir, err := ioutil.TempDir("", "")
		if err != nil {
			http.Error(wr, err.Error(), http.StatusInternalServerError)
			return
		}

		defer os.RemoveAll(tmpDir)
		imgPath := path.Join(tmpDir, "image.jpg")

		err = dcsvc.CaptureStill(imgPath)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusInternalServerError)
			return
		}

		imgBytes, err := ioutil.ReadFile(imgPath)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusInternalServerError)
			return
		}

		wr.Header().Set("Content-Type", "image/jpeg")
		_, err = wr.Write(imgBytes)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}