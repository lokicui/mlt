package taginfo

import (
	"bufio"
	"fmt"
	"github.com/astaxie/beego/logs"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TagInfo struct {
	Tid    int
	Status int
	Name   string
	Aliase string
	Icon   string
	Pid    int
	PName  string
}

var (
	gTidToTagInfoMap   = &map[int]*TagInfo{}
	gTNameToTagInfoMap = &map[string]*TagInfo{}
	gLastUpdateTime    time.Time
	gMutex             = new(sync.Mutex)
)

func Init(fname string) {
	ticker := time.NewTicker(time.Second * 1)
	go func() {
		for range ticker.C {
			ReloadConfigByModTime(fname)
		}
	}()
}

func GetTagInfoById(tid int) (info *TagInfo, ok bool) {
	info, ok = (*gTidToTagInfoMap)[tid]
	return info, ok
}

func GetTagInfoByName(tname string) (info *TagInfo, ok bool) {
	info, ok = (*gTNameToTagInfoMap)[tname]
	return info, ok
}

func ReloadConfigByModTime(fname string) {
	tidToTagInfoMap := map[int]*TagInfo{}
	tnameToTagInfoMap := map[string]*TagInfo{}
	finfo, err := os.Stat(fname)
	if err != nil {
		logs.Warn(fmt.Sprintf("fname=%s", fname), err)
	}
	if gLastUpdateTime == finfo.ModTime() {
		logs.Debug(fmt.Sprintf("fname=%s did't change", fname), " no need update")
		return
	} else {
		logs.Debug(fmt.Sprintf("fname=%s", fname), " update start ...")
		gLastUpdateTime = finfo.ModTime()
	}
	file, err := os.Open(fname)
	if err != nil {
		logs.Warn(fmt.Sprintf("fname=%s", fname), err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer([]byte{}, bufio.MaxScanTokenSize*100)
	for scanner.Scan() {
		line := scanner.Text()
		items := strings.Split(line, "\t")
		if len(items) < 9 {
			continue
		}
		tidstr := items[0]
		tid, err := strconv.Atoi(tidstr)
		if err != nil {
			logs.Warn(tidstr, " tidstr is not digit")
			continue
		}
		statustr := items[1]
		status, err := strconv.Atoi(statustr)
		if err != nil {
			logs.Warn(statustr, " statustr is not digit")
			continue
		}
		name := items[2]
		icon := items[3]
		aliase := items[4]
		_ = items[5] //tid
		_ = items[6] //name
		pidstr := items[7]
		pid, err := strconv.Atoi(pidstr)
		if err != nil {
			logs.Warn(pidstr, " pidstr is not digit")
			continue
		}
		pname := items[8]
		taginfo := &TagInfo{
			Tid:    tid,
			Status: status,
			Name:   name,
			Aliase: aliase,
			Icon:   icon,
			Pid:    pid,
			PName:  pname,
		}
		tidToTagInfoMap[tid] = taginfo
		tnameToTagInfoMap[name] = taginfo
	}
	if err := scanner.Err(); err != nil {
		logs.Warn(err)
	}
	gMutex.Lock()
	gTidToTagInfoMap = &tidToTagInfoMap
	gTNameToTagInfoMap = &tnameToTagInfoMap
	gMutex.Unlock()
	logs.Debug(fmt.Sprintf("fname=%s", fname), " update finished")
}
