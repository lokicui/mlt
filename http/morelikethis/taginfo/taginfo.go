package taginfo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/lokicui/mlt/g"
	"github.com/lokicui/mlt/http/morelikethis/segmenter"
	"github.com/lokicui/mlt/http/morelikethis/trie"
	"golang.org/x/net/context"
	elastic "gopkg.in/olivere/elastic.v5"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	UPDATE_INTERVAL_SECONDS int = 300
)

type TagInfo struct {
	Tid                int
	Status             int
	Name               string
	Aliase             string
	Icon               string
	Pid                int
	PName              string
	TopicFunctionality int //==1事件追踪 ==2邀请码
}

var (
	gTidToTagInfoMap   = &map[int]*TagInfo{}
	gTNameToTagInfoMap = &map[string]*TagInfo{}
	gTrie              = trie.New() //不分词，以字为纬度建立的trie && 分词后以词为纬度建立trie
	gMutex             = new(sync.Mutex)
	gLastUpdateTime    time.Time
)

func Init(fname string) {
	addrs := make([]string, 0, 16)
	for _, addr := range strings.Split(*g.ESTagInfoAddrs, ",") {
		addrs = append(addrs, addr)
	}
	client, err := elastic.NewClient(elastic.SetURL(addrs...), elastic.SetBasicAuth(*g.ESUser, *g.ESPasswd), elastic.SetSniff(false))
	if err != nil {
		logs.Critical(err)
	}
	//ReloadConfigByModTime(fname)
	ReloadConfigFromES(client)
	ticker := time.NewTicker(time.Second * time.Duration(UPDATE_INTERVAL_SECONDS))
	go func() {
		for range ticker.C {
			//ReloadConfigByModTime(fname)
			ReloadConfigFromES(client)
		}
	}()
}

func GetTagIDsByTF(topicFunctionality int) (tids []int) {
	for tid, info := range *gTidToTagInfoMap {
		if info.TopicFunctionality == topicFunctionality {
			tids = append(tids, tid)
		}
	}
	return
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
	for i := 0; i < len(keyPieces)-1; i++ {
		v := gTrie.Search(keyPieces[i:len(keyPieces)])
		if v != nil && v.GetValue() != nil {
			info := v.GetValue().(*TagInfo)
			infos = append(infos, info)
		}
	}
	return infos
}

func ReloadConfigFromES(client *elastic.Client) {
	tidToTagInfoMap := map[int]*TagInfo{}
	tnameToTagInfoMap := map[string]*TagInfo{}
	trie := trie.New()
	filterBoolQuery := elastic.NewBoolQuery()
	filterBoolQuery = filterBoolQuery.Must(elastic.NewTermQuery("show_status", 1))
	svc := client.Scroll("taginfo").Query(filterBoolQuery)
	for {
		searchResult, err := svc.Do(context.TODO())
		if err == io.EOF { // or err == io.EOF
			break
		}
		if err != nil {
			logs.Warn(err)
		}
		if searchResult == nil {
			logs.Warn("expected results != nil; got nil")
		}

		for _, hit := range searchResult.Hits.Hits {
			item := make(map[string]interface{})
			err := json.Unmarshal(*hit.Source, &item)
			if err != nil {
				logs.Error(err)
			}
			logs.Info(item)
			tid, err := strconv.Atoi(hit.Id)
			if err != nil {
				logs.Warn(fmt.Sprintf("tid=%#v illegal", hit.Id))
				continue
			}
			if _, ok := item["show_status"]; !ok {
				logs.Warn(hit.Id, " show_status=%s not exists")
				continue
			}
			if _, ok := item["tag_name"]; !ok {
				logs.Warn(hit.Id, " key=tag_name not exists")
				continue
			}
			if _, ok := item["alias_list"]; !ok {
				logs.Warn(hit.Id, " key=alias_list not exists")
				continue
			}
			if _, ok := item["icon_list"]; !ok {
				logs.Warn(hit.Id, " key=icon_list not exists")
				continue
			}
			if _, ok := item["topic_functionality"]; !ok {
				logs.Warn(hit.Id, " key=topic_functionality not exists")
				continue
			}
			showStatusStr, _ := item["show_status"].(string)
			showStatus, err := strconv.Atoi(showStatusStr)
			if err != nil {
				logs.Warn(hit.Id, fmt.Sprintf(" show_status=%#v illegal", item["show_status"]))
				continue
			}
			topicFunctionalityStr, _ := item["topic_functionality"].(string)
			topicFunctionality, err := strconv.Atoi(topicFunctionalityStr)
			if err != nil {
				logs.Warn(hit.Id, fmt.Sprintf(" topic_functionality=%#v illegal", item["topic_functionality"]))
				continue
			}
			name, _ := item["tag_name"].(string)
			aliase, _ := item["alias_list"].(string)
			icon, _ := item["icon_list"].(string)
			taginfo := &TagInfo{
				Tid:                tid,
				Status:             showStatus,
				Name:               name,
				Aliase:             aliase,
				Icon:               icon,
				Pid:                0,
				PName:              "",
				TopicFunctionality: topicFunctionality,
			}
			logs.Info(fmt.Sprintf("%#v", taginfo))
			//对name和aliase分词建立trie
			tidToTagInfoMap[taginfo.Tid] = taginfo
			tnameToTagInfoMap[taginfo.Name] = taginfo
			nameWords := Segment(taginfo.Name, false)
			aliaseWords := Segment(taginfo.Aliase, false)
			trie.Insert(nameWords, taginfo)
			trie.Insert(aliaseWords, taginfo)
		}
	}
	gMutex.Lock()
	gTidToTagInfoMap = &tidToTagInfoMap
	gTNameToTagInfoMap = &tnameToTagInfoMap
	gTrie = trie
	gMutex.Unlock()
	logs.Debug(len(tidToTagInfoMap), " tags update finished")
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
		tidToTagInfoMap[taginfo.Tid] = taginfo
		tnameToTagInfoMap[taginfo.Name] = taginfo
		nameWords := Segment(taginfo.Name, false)
		aliaseWords := Segment(taginfo.Aliase, false)
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
