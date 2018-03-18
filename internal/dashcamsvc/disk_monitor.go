package dashcamsvc

import (
	"time"
	"syscall"
	"github.com/sirupsen/logrus"
	"os"
	"sort"
	"io/ioutil"
	"path"
)

type recordings []os.FileInfo

func (p recordings) Len() int {
	return len(p)
}

func (p recordings) Less(i, j int) bool {
	return p[i].ModTime().Before(p[j].ModTime())
}

func (p recordings) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}


type DiskMonitor struct {
	stop chan struct{}
	path string
	usageTarget float64
}

func NewDiskMonitor(path string, usageTarget float64) *DiskMonitor {
	return &DiskMonitor{
		stop: make(chan struct{}),
		path: path,
		usageTarget: usageTarget,
	}
}

func (m *DiskMonitor) Quit() {
	logrus.Info("DiskMonitor stopping")
	close(m.stop)
}

func (m *DiskMonitor) Run() {
	for {
		select {
		case <- m.stop:
			break
		default:
			used, err := m.getUsage()
			if err != nil {
				logrus.Error(err)
			}

			logrus.Infof("DiskMonitor current usage %.2f", used)

			if used > m.usageTarget {
				m.rotateRecordings()
			}

			time.Sleep(60 * time.Second)
		}
	}
}

func (m *DiskMonitor) getUsage() (used float64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(m.path, &stat); err != nil {
		return 0, err
	}

	total := float64(stat.Blocks * uint64(stat.Bsize))
	available := float64(stat.Bavail * uint64(stat.Bsize))

	return (1 - (available / total)) * 100, nil
}

func (m *DiskMonitor) rotateRecordings() {
	files, err := ioutil.ReadDir(m.path)
	if err != nil {
		logrus.Panic(err)
	}

	recordings := recordings(files)
	sort.Sort(recordings)

	for _, recording := range recordings {
		logrus.Infof("DiskMonitor removing %s", recording.Name())
		err := os.Remove(path.Join(m.path, recording.Name()))
		if err != nil {
			logrus.Error(err)
			break
		}

		used, err := m.getUsage()
		if err != nil {
			logrus.Error(err)
			break
		}

		if used < m.usageTarget {
			break
		}
	}
}