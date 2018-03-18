package dashcamsvc

import (
	"fmt"
	"time"
	"os/exec"
	"os"
	"io/ioutil"
	"path"
	"github.com/sirupsen/logrus"
)

type command string
const (
	record command = "RECORD"
	stop command = "STOP"
	quit command = "QUIT"
)

type DashCamService struct {
	cmdChan chan command
	indicator *Indicator
	path string
}

func NewDashCamService(indicator *Indicator, path string) *DashCamService {
	return &DashCamService{
		cmdChan: make(chan command),
		indicator: indicator,
		path: path,
	}
}

func (d *DashCamService) StartRecording() {
	d.cmdChan <- record
}

func (d *DashCamService) StopRecording() {
	d.cmdChan <- stop
}

func (d *DashCamService) Quit() {
	d.cmdChan <- quit
}

func (d *DashCamService) Run(){
	running:
	for {
		select {
		case cmd := <- d.cmdChan:
			switch cmd {
			case record:
				logrus.Info("Service recording")
				d.indicator.Illuminate()
				recording:
				for {
					select {
					case cmd := <- d.cmdChan:
						switch cmd {
						case stop:
							logrus.Info("Service stopping")
							d.indicator.Extinguish()
							break recording
						case quit:
							logrus.Info("Service quitting")
							break running
						}
					default:
						d.record()
					}
				}
			case quit:
				break running
			}
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func (d *DashCamService) CaptureVideo(path string) (error) {
	logrus.Debugf("Capturing video to %s", path)
	err := exec.Command("raspivid", "-o", path, "-t", "300000", "-w", "1296", "-h", "972", "-fps", "30").Run()
	if err != nil {
		return err
	}

	return nil
}

func (d *DashCamService) CaptureStill(path string) error {
	logrus.Debugf("Capturing still to %s", path)
	err := exec.Command("raspistill", "-o", path).Run()
	if err != nil {
		return err
	}

	return nil
}


func (d *DashCamService) record() {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		logrus.Error(err)
		return
	}

	h264Path := path.Join(tmpDir, "video.h264")
	err = d.CaptureVideo(h264Path)
	if err != nil {
		logrus.Error(err)
		return
	}

	go func() {
		defer os.RemoveAll(tmpDir)

		mp4Path := path.Join(tmpDir, "video.mp4")
		err := d.convertToMP4(h264Path, mp4Path)
		if err != nil {
			logrus.Error(err)
			return
		}

		outputFilePath := fmt.Sprintf("%s%s.mp4", d.path, time.Now().Format(time.RFC3339))
		err = d.moveFile(mp4Path, outputFilePath)
		if err != nil {
			logrus.Error(err)
			return
		}
	}()
}

func (d *DashCamService) convertToMP4(inputPath string, outputPath string) error {
	logrus.Debugf("Converting %s to MP4 %s", inputPath, outputPath)
	err := exec.Command("MP4Box", "-add", inputPath, outputPath).Run()
	if err != nil {
		return err
	}

	return nil
}

func (d *DashCamService) moveFile(inputPath string, outputPath string) error {
	logrus.Debugf("Moving %s to %s", inputPath, outputPath)
	err := exec.Command("mv", inputPath, outputPath).Run()
	if err != nil {
		return err
	}

	return nil
}