package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/lokicui/mlt/g"
	"github.com/wangbin/jiebago/posseg"
	"golang.org/x/net/context"
	elastic "gopkg.in/olivere/elastic.v5"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var gStime time.Time = time.Now()
var gCnt uint64 = 0
var gLog *log.Logger = nil
var gAddr = flag.String("addr", "0.0.0.0", "Specify local addr for remote connects")
var gPort = flag.Int("port", 8080, "Specify local port for remote connects")
var gDebug = flag.Int("debug", 0, "debug level, 1 for debug")
var gVersion = flag.Bool("v", false, "show version")
var gSeg posseg.Segmenter

func init() {
	gLog = log.New(os.Stdout, "", log.Ldate|log.Lmicroseconds|log.Lshortfile)
	err := gSeg.LoadDictionary("conf/dict.txt")
	if err != nil {
		log.Fatal(err)
	}
	err = gSeg.LoadUserDictionary("conf/userdict.txt")
	if err != nil {
		log.Fatal(err)
	}
}

func GetHitData(query, UUID string) (hintArray []string) {
	url := fmt.Sprintf("http://hint.wenwen.sogou.com/web?uuid=%s&ie=utf8&callback=hintdata&src=wenwen.xgzs&query=%s",
		UUID,
		query)
	resp, err := http.Get(url)
	if err != nil {
		gLog.Printf("get hit data failed with:%s\n", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		gLog.Printf("read hit data failed with:%s\n", err)
	}
	jsonstr := strings.TrimSpace(string(body))
	if len(jsonstr) > 9 {
		jsonstr = jsonstr[len("hintdata(") : len(jsonstr)-1]
	}
	err = json.Unmarshal([]byte(jsonstr), &hintArray)
	if err != nil {
		gLog.Printf("hit data decode failed with:%s\n", err)
	}
	//fmt.Println(hintArray)
	return
}

func DebugQueryDSL(q elastic.Query) {
	src, err := q.Source()
	if err != nil {
		gLog.Fatal(err)
	}
	data, err := json.MarshalIndent(src, "", "    ")
	if err != nil {
		gLog.Fatal(err)
	}
	fmt.Println(string(data))
}

func getMustQueryWords(wordItems []WordInfo) (words []string) {
	for _, item := range wordItems {
		if item.Weight > 1 || item.Term_NImps <= 10 { //非必留词 or 重要度太低的词
			continue
		}
		words = append(words, SBC2DBC(item.Word))
	}
	return words
}

func MoreLikeThisQuery(request *MltRequest, client *elastic.Client) (result []interface{}, total int64, err error) {
	query := request.Query
	UUID := request.UUID
	stime := time.Now()
	atomic.AddUint64(&gCnt, 1)
	//opsFinal := atomic.LoadUint64(&gCnt)
	//if opsFinal%10000 == 0 {
	//	fmt.Printf("%s %d title query completed, req_rate:%.3f\n",
	//		time.Now(), opsFinal,
	//		float64(opsFinal)/float64(time.Since(gStime))*float64(time.Second))
	//}
	hintArray := GetHitData(query, UUID)
	qItems, err := SegmentQuery(query, false)
	if err != nil {
		gLog.Printf("uuid=%s,query=%s SegmentQuery failed with:%s\n", UUID, query, err)
		return
	}
	if len(qItems) == 0 {
		gLog.Printf("uuid=%s,query=%s SegmentQuery failed with len(qItems)=0\n", UUID, query)
		return
	}
	_ = time.Since(stime)
	lcssMap := make(map[string]float32)
	queryWords := getMustQueryWords(qItems)
	if *gDebug == 1 {
		fmt.Println(query)
	}
	for i, hintq := range hintArray {
		if i == 5 {
			break
		}
		items, err := SegmentQuery(hintq, false)
		if err != nil {
			continue
		}
		words := getMustQueryWords(items)
		subStringArray := GetLongestSubString(queryWords, words)
		subSequenceArray := GetLongestSubSequence(queryWords, words)
		if *gDebug == 1 {
			fmt.Println(hintq)
			fmt.Println("\t", queryWords, words, subStringArray, subSequenceArray)
		}
		lcss := strings.Join(subStringArray, " ")
		award := 1.0
		if i == 0 {
			award = 1.5 //首条结果给与一定加权
		}
		lcssMap[lcss] += float32(award)
	}
	////query用jieba分词分一下
	//queryWords := []string{}
	//for item:= range gSeg.Cut(query, false) {
	//    text := item.Text()
	//    pos := item.Pos()
	//    if strings.HasPrefix(pos, "x") { //标点符号
	//        continue
	//    }
	//    queryWords = append(queryWords, text)
	//}
	////hint的前N条结果进行jieba分词
	////计算query和hint前N条结果的最长公共子串
	//lcssMap := make(map[string]float32)
	//wordCnt := make(map[string]int)
	//for i, hintq := range hintArray {
	//    if i == 4 {
	//        break
	//    }
	//    wds := []string{}
	//    for item := range gSeg.Cut(hintq, false) {
	//        text := item.Text()
	//        pos := item.Pos()
	//        if strings.HasPrefix(pos, "x") { //标点符号
	//            continue
	//        }
	//        wds = append(wds, text)
	//        wordCnt[text] += 1
	//    }
	//    lcssArray := GetLongestSubstring(queryWords, wds)
	//    if (*gDebug == 1) {
	//        fmt.Println(queryWords, wds, lcssArray)
	//    }
	//    lcss := strings.Join(lcssArray, "")
	//    award := 1.0
	//    if i == 0 {
	//        award = 1.5 //首条结果给与一定加权
	//    }
	//    lcssMap[lcss] += float32(award)
	//}
	//convert lcssMap -> lcss tuple
	type Pair struct {
		key   string
		value float32
	}
	lcss2WeightArray := make([]Pair, 0, len(lcssMap))
	for k, v := range lcssMap {
		lcss2WeightArray = append(lcss2WeightArray, Pair{key: k, value: v})
	}
	sort.Slice(lcss2WeightArray, func(i, j int) bool {
		return lcss2WeightArray[i].value > lcss2WeightArray[j].value
	})

	//sogou 分词的重要度
	sortedWords := make([]WordInfo, len(qItems))
	copy(sortedWords, qItems)
	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[i].Term_NImps > sortedWords[j].Term_NImps
	})
	if *gDebug == 1 {
		for i, item := range sortedWords {
			fmt.Printf("%d %#v %d\n", i, item.Word, item.Term_NImps)
		}
	}
	importantKwds := ""
	if len(queryWords) > 5 {
		for _, item := range lcss2WeightArray {
			if len(importantKwds) == 0 {
				if len([]rune(item.key)) > 2 || len([]rune(item.key)) == 2 && item.value > 4 { //公共字串太短的不考虑
					importantKwds = item.key
				}
			}
			if *gDebug == 1 {
				fmt.Println(item.key, item.value)
			}
		}
	}

	likeTextCntThreshold := 3
	likeTextArray := append([]string{query}, hintArray...)
	if len(likeTextArray) > likeTextCntThreshold {
		likeTextArray = likeTextArray[:likeTextCntThreshold]
	}

	doctypes := make([]interface{}, 0, 3)
	indextypes := make([]string, 0, 2)
	for i := 0; i < 4; i++ {
		if (request.Typev & (1 << uint(i))) > 0 {
			if i < 3 {
				docType := i + 1
				doctypes = append(doctypes, docType)
				if len(indextypes) == 0 {
					indextypes = append(indextypes, "question")
				}
			}
			if i == 3 {
				indextypes = append(indextypes, "article")
			}
		}
	}

	//计算NImpmps最高的两个term 作为ES的term filter
	filterBoolQuery := elastic.NewBoolQuery()
	//filterBoolQuery := elastic.NewBoolQuery().MinimumNumberShouldMatch(1)
	//for i, item := range sortedWords {
	//	if i == 2 || item.Weight > 1{
	//		break
	//	}
	//	//filterBoolQuery = filterBoolQuery.Should(elastic.NewTermQuery("title", SBC2DBC(item.Word)))
	//}
	filterMustQuery := elastic.NewBoolQuery().MinimumNumberShouldMatch(1)
	filterMustQuery = filterMustQuery.Should(elastic.NewTermsQuery("type", doctypes...))
	filterMustQuery = filterMustQuery.Should(elastic.NewTermQuery("_type", "article"))
	filterBoolQuery = filterBoolQuery.Must(elastic.NewTermQuery("status", 2))
	filterBoolQuery = filterBoolQuery.Must(filterMustQuery)

	mltQuery := elastic.NewMoreLikeThisQuery().
		Field("title").
		MinTermFreq(1).
		MaxQueryTerms(20).
		MinDocFreq(1).
		MinimumShouldMatch("20%").
		LikeText(likeTextArray...)

	boolQuery := elastic.NewBoolQuery().
		Filter(filterBoolQuery).
		Must(mltQuery)

	if len([]rune(importantKwds)) > 0 {
		matchQuery := elastic.NewMatchQuery("title", importantKwds).Operator("or")
		boolQuery = boolQuery.Must(matchQuery)
	}

	for i, id := range request.Tids {
		if i == 1 {
			break
		}
		termQuery := elastic.NewTermQuery("tags", id).Boost(2.0)
		boolQuery = boolQuery.Should(termQuery)
	}
	//命中实体词的认为实体词是核心词, 必须全命中在这里强制
	//for item := range gSeg.Cut(query, false) {
	//    text := item.Text()
	//    pos := item.Pos()
	//    if pos == "entity" {
	//        matchQuery := elastic.NewMatchQuery("title", text).Operator("or")
	//        boolQuery = boolQuery.Must(matchQuery)
	//        break
	//    }
	//}
	if *gDebug == 1 {
		DebugQueryDSL(boolQuery)
	}
	//fmt.Println(indextypes, doctypes)
	//fs := elastic.NewFetchSourceContext(true).Include("title", "id")
	fs := elastic.NewFetchSourceContext(true)
	res, err := client.Search().
		Index("luedongshe").
		Type(indextypes...).
		From(request.Start).
		Size(request.Limit).
		Preference(request.Preference).
		Query(boolQuery).
		FetchSourceContext(fs).
		Timeout("150ms").
		Pretty(false).
		Do(context.TODO())
	if err != nil {
		gLog.Printf("client.search err with:%s\n", err)
		return
	}
	if res.Hits == nil {
		gLog.Print("expected SearchResult.Hits != nil; got nil")
		return
	}
	//res.TotalHits()
	total = res.TotalHits()
	//if res.Hits.TotalHits == 0
	if total == 0 {
		gLog.Printf("expected SearchResult.Hits.TotalHits > %d; got %d", 0, res.Hits.TotalHits)
		return
	}
	result = make([]interface{}, 0, len(res.Hits.Hits))
	for _, hit := range res.Hits.Hits {
		matchItem := make(map[string]interface{})
		matchItem["_index"] = hit.Index
		matchItem["_type"] = hit.Type
		matchItem["_id"] = hit.Id
		matchItem["_score"] = hit.Score
		item := make(map[string]interface{})
		err := json.Unmarshal(*hit.Source, &item)
		if err != nil {
			gLog.Printf("json unmarshal failed with:%s", err)
			continue
		}
		matchItem["_source"] = item
		result = append(result, matchItem)
	}
	return
}

func genRequest(r *http.Request) (request *MltRequest, retcode int, err error) {
	request = NewMltRequest()
	r.ParseForm()
	m := r.Form
	if value, ok := m["uuid"]; ok && len(value) > 0 {
		request.UUID = value[0]
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no uuid arg")
		gLog.Printf("%s\n", errmsg)
		err = errors.New(errmsg)
		return
	}
	if value, ok := m["pretty"]; ok && len(value) > 0 {
		if value[0] != "false" && value[0] != "0" {
			request.Pretty = true
		}
	}
	if value, ok := m["start"]; ok && len(value) > 0 {
		v, err := strconv.Atoi(value[0])
		if err == nil {
			request.Start = v
		}
	}
	if value, ok := m["limit"]; ok && len(value) > 0 {
		v, err := strconv.Atoi(value[0])
		if err == nil {
			request.Limit = v
		}
	}
	if value, ok := m["type"]; ok && len(value) > 0 {
		request.Typev, err = strconv.Atoi(value[0])
		if err != nil || request.Typev == 0 {
			retcode = 1
			errmsg := fmt.Sprintf("argument error failed with:%s", "type arg illegal or equals to zero")
			gLog.Printf("uuid:%s, %s\n", request.UUID, errmsg)
			err = errors.New(errmsg)
			return
		}
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no type arg")
		gLog.Printf("uuid:%s, %s\n", request.UUID, errmsg)
		err = errors.New(errmsg)
		return
	}
	if value, ok := m["preference"]; ok && len(value) > 0 {
		request.Preference = value[0]
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no preference arg")
		gLog.Printf("uuid:%s, %s\n", request.UUID, errmsg)
		err = errors.New(errmsg)
		return
	}
	//tids=tag ids, 打算用tagid召回一部分内容
	if value, ok := m["tids"]; ok && len(value) > 0 {
		tidsstr := strings.Split(value[0], ",")
		for _, tidstr := range tidsstr {
			v, err := strconv.Atoi(tidstr)
			if err == nil {
				request.Tids = append(request.Tids, v)
			}
		}
		sort.Sort(sort.IntSlice(request.Tids))
	}

	if value, ok := m["query"]; ok && len(value) > 0 {
		request.Query = value[0]
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no query arg")
		gLog.Printf("uuid:%s, %s\n", request.UUID, errmsg)
		err = errors.New(errmsg)
		return
	}
	return
}

func jsonHandler(w http.ResponseWriter, r *http.Request, client *elastic.Client) {
	stime := time.Now()
	request := NewMltRequest()
	retcode := 0
	result := []interface{}{}
	var total int64 = 0
	var err error = nil
	defer func() {
		errmsg := ""
		if err != nil {
			errmsg = err.Error()
		}
		took := float64(time.Since(stime)) / float64(time.Second)
		vmap := make(map[string]interface{})
		vmap["rawreq"] = r.URL
		vmap["retcode"] = retcode
		vmap["request"] = request
		vmap["took"] = took
		vmap["errmsg"] = errmsg
		vmap["data"] = result
		vmap["total"] = total
		vmap["retnum"] = len(result)
		vjson := []byte{'{', '}'}
		if request.Pretty {
			vjson, _ = json.MarshalIndent(vmap, "", "    ")
		} else {
			vjson, _ = json.Marshal(vmap)
		}
		fmt.Fprintf(w, "%s", vjson)
		delete(vmap, "data")
		logjson, _ := json.Marshal(vmap)
		gLog.Printf("%s", logjson)
	}()
	request, retcode, err = genRequest(r)
	if err != nil {
		retcode = 1
		return
	}
	result, total, err = MoreLikeThisQuery(request, client)
	if err != nil {
		retcode = 3
		errmsg := fmt.Sprintf("more_like_this query failed with:%s", err)
		gLog.Printf("uuid:%s, %s\n", request.UUID, errmsg)
		err = errors.New(errmsg)
		return
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, *elastic.Client)) http.HandlerFunc {
	//10.134.13.99
	client, err := elastic.NewClient(elastic.SetURL("http://10.134.13.99:9200", "http://10.134.14.27:9200", "http://10.134.28.85:9200"))
	if err != nil {
		// Handle error
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		jsonHandler(w, r, client)
	}
}

//func main() {
//	// Create a client and connect to http://192.168.2.10:9201
//	//client, err := elastic.NewClient(elastic.SetURL("http://10.134.29.127:9200", "http://10.134.53.116:9200", "http://10.134.96.106:9200"))
//	//client, err := elastic.NewClient(elastic.SetURL("http://10.134.96.50:9200", "http://10.134.96.51:9200", "http://10.134.96.52:9200"))
//	flag.Parse()
//	if *gVersion {
//		fmt.Println(g.VERSION)
//		os.Exit(0)
//	}
//	http.HandleFunc("/json/", makeHandler(jsonHandler))
//	err := http.ListenAndServe(*gAddr+":"+fmt.Sprintf("%d", *gPort), nil)
//	if err != nil {
//		gLog.Fatal(err)
//	}
//	gLog.Printf("%s finish req\n", time.Now())
//}

func main() {
	flag.Parse()
	if *gVersion {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}
	http.HandleFunc("/json/", makeHandler(jsonHandler))
	err := http.ListenAndServe(*gAddr+":"+fmt.Sprintf("%d", *gPort), nil)
	if err != nil {
		gLog.Fatal(err)
	}
	gLog.Printf("%s finish req\n", time.Now())
}
