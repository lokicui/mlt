package taginfo

import (
	"bufio"
	"fmt"
	"github.com/astaxie/beego/logs"
    "github.com/lokicui/mlt/http/morelikethis/segmenter"
    "github.com/lokicui/mlt/http/morelikethis/trie"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
    UPDATE_INTERVAL_SECONDS int = 10
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
    gTrie              = trie.New() //不分词，以字为纬度建立的trie && 分词后以词为纬度建立trie
	gMutex             = new(sync.Mutex)
	gLastUpdateTime      time.Time
)

func Init(fname string) {
    ReloadConfigByModTime(fname)
	ticker := time.NewTicker(time.Second * time.Duration(UPDATE_INTERVAL_SECONDS))
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

func Segment(str string, seg bool) (words []string) {
    if seg {
        for item := range segmenter.GetSegmenter().Cut(str, false) {
            text := item.Text()
            pos := item.Pos()
            if strings.HasPrefix(pos, "x") { //标点符号
                continue
            }
            words = append(words, text)
        }
    } else {
        for _, t := range []rune(str) {
            words = append(words, string(t))
        }
    }
    return words
}

func SearchTagInfoByName(tname string, seg bool) (infos []*TagInfo) {
    keyPieces := Segment(tname, seg)
    //longest search
    for i := 0; i < len(keyPieces) - 1; i ++ {
        v := gTrie.Search(keyPieces[i:len(keyPieces)])
        if v != nil && v.GetValue() != nil {
            info := v.GetValue().(*TagInfo)
            infos = append(infos, info)
        }
    }
    return infos
}

func ReloadConfigByModTime(fname string) {
	tidToTagInfoMap := map[int]*TagInfo{}
	tnameToTagInfoMap := map[string]*TagInfo{}
    trie := trie.New()
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
        if status == 0 || status > 16 {
            logs.Info(fmt.Sprintf("tid=%d,tname=%s,status=%d will not be imported!", tid, name, status))
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
        //对name和aliase分词建立trie
		tidToTagInfoMap[tid] = taginfo
		tnameToTagInfoMap[name] = taginfo
        nameWords := Segment(name, false)
        aliaseWords := Segment(aliase, false)
        trie.Insert(nameWords, taginfo)
        trie.Insert(aliaseWords, taginfo)
	}
	if err := scanner.Err(); err != nil {
		logs.Warn(err)
	}
	gMutex.Lock()
	gTidToTagInfoMap = &tidToTagInfoMap
	gTNameToTagInfoMap = &tnameToTagInfoMap
    gTrie = trie
	gMutex.Unlock()
	logs.Debug(fmt.Sprintf("fname=%s", fname), " update finished")
}
