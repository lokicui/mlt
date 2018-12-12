package morelikethis

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/lokicui/mlt/g"
	"github.com/lokicui/mlt/http/morelikethis/taginfo"
	"github.com/lokicui/mlt/utils"
	"golang.org/x/net/context"
	elastic "gopkg.in/olivere/elastic.v5"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	gESClient *elastic.Client = nil
)

func init() {
	addrs := make([]string, 0, 16)
	for _, addr := range strings.Split(*g.ESAddrs, ",") {
		addrs = append(addrs, addr)
	}
	client, err := elastic.NewClient(elastic.SetURL(addrs...), elastic.SetBasicAuth(*g.ESUser, *g.ESPasswd), elastic.SetSniff(false))
	if err != nil {
		logs.Critical(err)
	}
	gESClient = client
}

func GetHitData(query, UUID string) (hintArray []string) {
	query = strings.Replace(query, "\n", " ", -1)
	url := fmt.Sprintf("http://hint.wenwen.sogou.com/web?uuid=%s&ie=utf8&callback=hintdata&src=wenwen.xgzs&query=%s",
		UUID,
		query)

	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}
	resp, err := client.Get(url)
	if err != nil {
		logs.Warn(fmt.Printf("get hit data failed with:%s", err))
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Warn(fmt.Printf("read hit data failed with:%s", err))
	}
	jsonstr := strings.TrimSpace(string(body))
	if len(jsonstr) > 9 {
		jsonstr = jsonstr[len("hintdata(") : len(jsonstr)-1]
	}
	err = json.Unmarshal([]byte(jsonstr), &hintArray)
	if err != nil {
		logs.Debug(fmt.Printf("hit data decode failed with:%s", err))
	}
	//fmt.Println(hintArray)
	return
}

func DebugQueryDSL(q elastic.Query) {
	src, err := q.Source()
	if err != nil {
		logs.Critical(err)
	}
	data, err := json.MarshalIndent(src, "", "    ")
	if err != nil {
		logs.Critical(err)
	}
	fmt.Println(string(data))
}

func getMustQueryWords(wordItems []utils.WordInfo) (words []string) {
	for _, item := range wordItems {
		if item.Weight > 1 || (item.Term_NImps != 0 && item.Term_NImps <= 5) { //非必留词 or 重要度太低的词
			continue
		}
		words = append(words, utils.SBC2DBC(item.Word))
	}
	return words
}

func MoreLikeThisQuery(request *MltRequest, client *elastic.Client) (result []interface{}, TFIDFMap map[string]float32, total int64, err error) {
	TFIDFMap = make(map[string]float32)
	query := request.Query
	UUID := request.UUID
	stime := time.Now()
	hintArray := GetHitData(query, UUID)
	addrs := beego.AppConfig.Strings("SegmentServer")
	//fmt.Println(beego.AppConfig.Strings("SegmentServer"))
	qItems, err := utils.SegmentQuery(addrs, query, false)
	if err != nil {
		logs.Debug(fmt.Printf("uuid=%s,query=%s SegmentQuery failed with:%s", UUID, query, err))
		return
	}
	if len(qItems) == 0 {
		logs.Debug(fmt.Printf("uuid=%s,query=%s SegmentQuery failed with len(qItems)=0", UUID, query))
		return
	}
	_ = time.Since(stime)
	lcssMap := make(map[string]float32)
	weightMap := make(map[string]float32)
	indexToTerm := make(map[int]string)
	for i, item := range qItems {
		word := utils.SBC2DBC(item.Word)
		indexToTerm[i] = word
		weightMap[word] = float32(item.Term_NImps)
	}
	queryWords := getMustQueryWords(qItems)
	if request.Debug {
		fmt.Println(query)
	}
	for i, hintq := range hintArray {
		if i == 5 {
			break
		}
		items, err := utils.SegmentQuery(addrs, hintq, false)
		if err != nil {
			continue
		}
		words := getMustQueryWords(items)
		subStringArray := utils.GetLongestSubString(queryWords, words)
		for _, w := range subStringArray {
			TFIDFMap[w] += 1.0 * weightMap[w]
		}
		subSequenceArray := utils.GetLongestSubSequence(queryWords, words)
		for _, w := range subSequenceArray {
			TFIDFMap[w] += 1.0 * weightMap[w]
		}
		if request.Debug {
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

	term2WeightArray := make([]Pair, 0, len(TFIDFMap))
	for k, v := range TFIDFMap {
		term2WeightArray = append(term2WeightArray, Pair{key: k, value: v})
	}
	sort.Slice(term2WeightArray, func(i, j int) bool {
		return term2WeightArray[i].value > term2WeightArray[j].value
	})

	//sogou 分词的重要度
	sortedWords := make([]utils.WordInfo, len(qItems))
	copy(sortedWords, qItems)
	sort.Slice(sortedWords, func(i, j int) bool {
		return sortedWords[i].Term_NImps > sortedWords[j].Term_NImps
	})
	if request.Debug {
		for i, item := range sortedWords {
			fmt.Printf("%d %#v %d\n", i, item.Word, item.Term_NImps)
		}
		for k, v := range TFIDFMap {
			fmt.Printf("%s %.3f\n", k, v)
		}
	}
	importantKwdsArray := []string{}
	for i, item := range term2WeightArray {
		if i == 3 {
			break
		}
		importantKwdsArray = append(importantKwdsArray, item.key)
		if request.Debug {
			fmt.Println(item.key, item.value)
		}
	}
	importantKwds := strings.Join(importantKwdsArray, " ")
	_ = importantKwds
	//importantKwds := ""
	//if len(queryWords) > 5 {
	//	for _, item := range lcss2WeightArray {
	//		if len(importantKwds) == 0 {
	//			if len([]rune(item.key)) > 2 || len([]rune(item.key)) == 2 && item.value > 4 { //公共字串太短的不考虑
	//				importantKwds = item.key
	//			}
	//		}
	//		if request.Debug {
	//			fmt.Println(item.key, item.value)
	//		}
	//	}
	//}

	likeTextCntThreshold := 2
	likeTextArray := append([]string{query}, hintArray...)
	if len(likeTextArray) > likeTextCntThreshold {
		likeTextArray = likeTextArray[:likeTextCntThreshold]
	}

	doctypes := make([]interface{}, 0, 3)
	indextypes := make([]string, 0, 2)
	for i := 0; i < 4; i++ {
		if (request.Type & (1 << uint(i))) > 0 {
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
	//	//filterBoolQuery = filterBoolQuery.Should(elastic.NewTermQuery("title", utils.SBC2DBC(item.Word)))
	//}
	filterMustQuery := elastic.NewBoolQuery().MinimumNumberShouldMatch(1)
	filterMustQuery = filterMustQuery.Should(elastic.NewTermsQuery("type", doctypes...))

	//过滤出定向抓取系统导入的数据
	bo := elastic.NewBoolQuery()
	bo = bo.Must(elastic.NewTermQuery("_type", "article"))
	bo = bo.Must(elastic.NewTermQuery("refType", 2))      //新略懂app抓取系统导入的数据
	bo = bo.Must(elastic.NewTermQuery("listOpenType", 1)) //中间页

	bo1 := elastic.NewBoolQuery()
	bo1 = bo1.Must(elastic.NewTermQuery("_type", "article"))
	bo1 = bo1.MustNot(elastic.NewTermQuery("refType", 2))

	//filterMustQuery = filterMustQuery.Should(elastic.NewTermQuery("_type", "article"))
	filterMustQuery = filterMustQuery.Should(bo)
	filterMustQuery = filterMustQuery.Should(bo1)

	filterBoolQuery = filterBoolQuery.Must(elastic.NewTermQuery("status", 2))
	//if len([]rune(importantKwds)) > 0 {
	//	matchQuery := elastic.NewMatchQuery("title", importantKwds).Operator("or")
	//	filterBoolQuery = filterBoolQuery.Must(matchQuery)
	//}

	//   origin       来源
	//1       群问问个人
	//2       QQ群
	//3       主站团儿
	//4       略懂社/主站试水(焦点问答)
	//5       第一类开放平台
	//6       群友圈(通过群友圈评论回答或帖子)
	//100     略懂app
	//101     哥伦布后台人工添加
	//102     微信小程序
	//103     哥伦布后台机器灌入
	//104     搜索APP
	//105     搜狗阅读
	//106     头条阅读(demo)
	//1000        darwin主站
	filterBoolQuery = filterBoolQuery.MustNot(elastic.NewTermQuery("origin", 103))
	tids := taginfo.GetTagIDsByTF(2) // == 2 邀请码的数据过滤掉
	s := make([]interface{}, len(tids))
	for i, v := range tids {
		s[i] = v
	}
	filterBoolQuery = filterBoolQuery.MustNot(elastic.NewTermsQuery("tags", s...))
	filterBoolQuery = filterBoolQuery.Must(filterMustQuery)

	mltQuery := elastic.NewMoreLikeThisQuery().
		Field("title", "simpleContent"). //为了兼容略懂article某些内容只有content(就是simpleContent)没有title
		MinTermFreq(1).
		MaxQueryTerms(20).
		MinDocFreq(1).
		MinimumShouldMatch("20%").
		LikeText(likeTextArray...)

	boolQuery := elastic.NewBoolQuery().
		Filter(filterBoolQuery).
		Must(mltQuery)

	//if len([]rune(importantKwds)) > 0 {
	//	matchQuery := elastic.NewMatchQuery("title", importantKwds).Operator("or")
	//	boolQuery = boolQuery.Must(matchQuery)
	//}

	for i, id := range request.TidsList {
		if i == 1 {
			break
		}
		termQuery := elastic.NewTermQuery("tags", id).Boost(1.2)
		boolQuery = boolQuery.Should(termQuery)
	}
	if request.Debug {
		DebugQueryDSL(boolQuery)
	}
	//fmt.Println(indextypes, doctypes)
	//fs := elastic.NewFetchSourceContext(true).Include("title", "id")
	fs := elastic.NewFetchSourceContext(true).Exclude("answers")
	//fs := elastic.NewFetchSourceContext(true)
	ctx, cancel := context.WithTimeout(context.Background(), 160*time.Millisecond)
	defer cancel()
	res, err := client.Search().
		Index("luedongshe").
		Type(indextypes...).
		From(request.Start).
		Size(request.Limit).
		Preference(request.Preference).
		Query(boolQuery).
		FetchSourceContext(fs).
		Timeout("150ms").
		Pretty(request.Pretty).
		Do(ctx)
	if err != nil {
		logs.Debug(fmt.Printf("client.search err with:%s", err))
		return
	}
	if res.Hits == nil {
		logs.Debug("expected SearchResult.Hits != nil; got nil")
		return
	}
	//res.TotalHits()
	total = res.TotalHits()
	//if res.Hits.TotalHits == 0
	if total == 0 {
		logs.Debug("expected SearchResult.Hits.TotalHits > %d; got 0")
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
			logs.Debug(fmt.Printf("json unmarshal failed with:%s", err))
			continue
		}
		matchItem["_source"] = item
		result = append(result, matchItem)
	}
	return
}

func GenGetByTidRequest(m url.Values) (request *GetByTidRequest, retcode int, err error) {
	request = NewGetByTidRequest()
	if value, ok := m["uuid"]; ok && len(value) > 0 {
		request.UUID = value[0]
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no uuid arg")
		logs.Debug(errmsg)
		err = errors.New(errmsg)
		return
	}
	if value, ok := m["pretty"]; ok && len(value) > 0 {
		if value[0] != "false" && value[0] != "0" {
			request.Pretty = true
		}
	}
	if value, ok := m["debug"]; ok && len(value) > 0 {
		if value[0] != "false" && value[0] != "0" {
			request.Debug = true
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
		request.Type, err = strconv.Atoi(value[0])
		if err != nil || request.Type == 0 {
			retcode = 1
			errmsg := fmt.Sprintf("argument error failed with:%s", "type arg illegal or equals to zero")
			logs.Debug(fmt.Printf("uuid:%s, %s", request.UUID, errmsg))
			err = errors.New(errmsg)
			return
		}
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no type arg")
		logs.Debug(fmt.Printf("uuid:%s, %s", request.UUID, errmsg))
		err = errors.New(errmsg)
		return
	}
	if value, ok := m["preference"]; ok && len(value) > 0 {
		request.Preference = value[0]
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no preference arg")
		logs.Debug(fmt.Printf("uuid:%s, %s", request.UUID, errmsg))
		err = errors.New(errmsg)
		return
	}
	if value, ok := m["tid"]; ok && len(value) > 0 {
		request.Tid = value[0]
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no Tid arg")
		logs.Debug(fmt.Printf("uuid:%s, %s", request.UUID, errmsg))
		err = errors.New(errmsg)
		return
	}
	return
}

func GenMltRequest(m url.Values) (request *MltRequest, retcode int, err error) {
	request = NewMltRequest()
	if value, ok := m["uuid"]; ok && len(value) > 0 {
		request.UUID = value[0]
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no uuid arg")
		logs.Debug(errmsg)
		err = errors.New(errmsg)
		return
	}
	if value, ok := m["pretty"]; ok && len(value) > 0 {
		if value[0] != "false" && value[0] != "0" {
			request.Pretty = true
		}
	}
	if value, ok := m["debug"]; ok && len(value) > 0 {
		if value[0] != "false" && value[0] != "0" {
			request.Debug = true
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
		request.Type, err = strconv.Atoi(value[0])
		if err != nil || request.Type == 0 {
			retcode = 1
			errmsg := fmt.Sprintf("argument error failed with:%s", "type arg illegal or equals to zero")
			logs.Debug(fmt.Printf("uuid:%s, %s", request.UUID, errmsg))
			err = errors.New(errmsg)
			return
		}
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no type arg")
		logs.Debug(fmt.Printf("uuid:%s, %s", request.UUID, errmsg))
		err = errors.New(errmsg)
		return
	}
	if value, ok := m["preference"]; ok && len(value) > 0 {
		request.Preference = value[0]
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no preference arg")
		logs.Debug(fmt.Printf("uuid:%s, %s", request.UUID, errmsg))
		err = errors.New(errmsg)
		return
	}
	//tids=tag ids, 打算用tagid召回一部分内容
	if value, ok := m["tids"]; ok && len(value) > 0 {
		request.Tids = value[0]
		tidsstr := strings.Split(request.Tids, ",")
		for _, tidstr := range tidsstr {
			v, err := strconv.Atoi(tidstr)
			if err == nil {
				request.TidsList = append(request.TidsList, v)
			}
		}
		sort.Sort(sort.IntSlice(request.TidsList))
	}

	if value, ok := m["query"]; ok && len(value) > 0 {
		request.Query = value[0]
	} else {
		retcode = 1
		errmsg := fmt.Sprintf("argument error failed with:%s", "no query arg")
		logs.Debug(fmt.Printf("uuid:%s, %s", request.UUID, errmsg))
		err = errors.New(errmsg)
		return
	}
	return
}

func moreLikeThisHandler(w http.ResponseWriter, r *http.Request, client *elastic.Client) {
	stime := time.Now()
	r.ParseForm()
	request, retcode, err := GenMltRequest(r.Form)
	if err != nil {
		retcode = 1
	}
	result, keywordsMap, total, err := MoreLikeThisQuery(request, client)
	if err != nil {
		retcode = 3
		errmsg := fmt.Sprintf("more_like_this query failed with:%s", err)
		logs.Debug(fmt.Printf("uuid:%s, %s", request.UUID, errmsg))
		err = errors.New(errmsg)
	}
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
	vmap["keywords"] = keywordsMap
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
	logs.Debug(string(logjson))
}

func GetMoreLikeThisResult(request *MltRequest) (result []interface{}, keywordsMap map[string]float32) {
	query := request.Query
	hitTagInfos := taginfo.SearchTagInfoByName(query, false)
	for i, hitTagInfo := range hitTagInfos {
		logs.Debug(fmt.Sprintf("%d-hitTagName=%#v", i, hitTagInfo))
	}
	result, keywordsMap, _, _ = MoreLikeThisQuery(request, gESClient)
	return result, keywordsMap
}

func GetByTidResult(request *GetByTidRequest) (result []interface{}) {
	doctypes := make([]interface{}, 0, 3)
	indextypes := make([]string, 0, 2)
	for i := 0; i < 4; i++ {
		if (request.Type & (1 << uint(i))) > 0 {
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

	mustQuery := elastic.NewBoolQuery().MinimumNumberShouldMatch(1)
	mustQuery = mustQuery.Should(elastic.NewTermsQuery("type", doctypes...))
	mustQuery = mustQuery.Should(elastic.NewTermQuery("_type", "article"))
	boolQuery := elastic.NewBoolQuery()
	boolQuery = boolQuery.Must(elastic.NewTermQuery("status", 2))
	boolQuery = boolQuery.Must(elastic.NewTermQuery("tags", request.Tid))
	boolQuery = boolQuery.MustNot(elastic.NewTermQuery("origin", 13))
	boolQuery = boolQuery.Must(mustQuery)

	if request.Debug {
		DebugQueryDSL(boolQuery)
	}
	//fmt.Println(indextypes, doctypes)
	//fs := elastic.NewFetchSourceContext(true).Include("title", "id")
	fs := elastic.NewFetchSourceContext(true)
	ctx, cancel := context.WithTimeout(context.Background(), 160*time.Millisecond)
	defer cancel()
	res, err := gESClient.Search().
		Index("luedongshe").
		Type(indextypes...).
		From(request.Start).
		Size(request.Limit).
		Preference(request.Preference).
		Query(boolQuery).
		FetchSourceContext(fs).
		Timeout("150ms").
		Pretty(request.Pretty).
		Do(ctx)
	if err != nil {
		logs.Debug(fmt.Printf("client.search err with:%s", err))
		return
	}
	if res.Hits == nil {
		logs.Debug("expected SearchResult.Hits != nil; got nil")
		return
	}
	//res.TotalHits()
	total := res.TotalHits()
	//if res.Hits.TotalHits == 0
	if total == 0 {
		logs.Debug("expected SearchResult.Hits.TotalHits > %d; got 0")
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
			logs.Debug(fmt.Printf("json unmarshal failed with:%s", err))
			continue
		}
		matchItem["_source"] = item
		result = append(result, matchItem)
	}
	return
}

func MakeHandler(fn func(http.ResponseWriter, *http.Request, *elastic.Client)) http.HandlerFunc {
	//10.134.13.99
	//client, err := elastic.NewClient(elastic.SetURL("http://10.134.13.99:9200", "http://10.134.14.27:9200", "http://10.134.28.85:9200"))
	//if err != nil {
	//	// Handle error
	//	//panic(err)
	//	logs.Critical(err)
	//}
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, gESClient)
	}
}
