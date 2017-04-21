package main

import (
    "fmt"
    "log"
    "bufio"
    "sync"
    "strings"
    "regexp"
    //iconv "github.com/djimenez/iconv-go"
    _ "strconv"
    "encoding/xml"
    "encoding/json"
    _ "io/ioutil"
    "html"
    "os"
)

type TAnswer struct {
    XMLName         xml.Name            `json:"-" xml:"answer"`
    Type            string              `json:"type" xml:"type"`
    ID              int64               `json:"id" xml:"id"`
    Uin             string              `json:"uin" xml:"uin"`
    Unick           string              `json:"unick" xml:"unick"`
    Anonymous       string              `json:"anonymous" xml:"anonymous"`
    Ip              string              `json:"ip" xml:"ip"`
    Rank            string              `json:"rank" xml:"rank"`
    Role            string              `json:"role" xml:"role"`
    Qunid           string              `json:"qunid" xml:"qunid"`
    Content         string              `json:"content" xml:"content"`
    Time            string              `json:"time" xml:"time"`
    Star            string              `json:"star" xml:"star"`
    Elapse          string              `json:"elapse" xml:"elapse"`
    UserLevel       string              `json:"userLevel" xml:"userLevel"`
    PicURL          string              `json:"picURL" xml:"picURL"`
    AdoptRate       string              `json:"adoptRate" xml:"adoptRate"`
    Rich            string              `json:"rich" xml:"rich"`
    RichText        string              `json:"richText" xml:"richText"`
    Orig            string              `json:"orig" xml:"orig"`
    Up              string              `json:"up" xml:"up"`
    Down            string              `json:"down" xml:"down"`
    Follow          string              `json:"follow" xml:"follow"`
    Thank           string              `json:"thank" xml:"thank"`
}

type TAnswers struct {
    Answers         []TAnswer           `json:"answers" xml:"answers"`
}

type TQuestion struct {
    XMLName         xml.Name            `json:"-" xml:"question"`
    Qid             int64               `json:"qid" xml:"qid"`
    State           string              `json:"state" xml:"state"`
    Property        string              `json:"property" xml:"property"`
    Rich            string              `json:"rich" xml:"rich"`
    Orig            string              `json:"orig" xml:"orig"`
    Title           string              `json:"title" xml:"title"`
    Description     string              `json:"description" xml:"description"`
    CategoryID      string              `json:"categoryID" xml:"categoryID"`
    AskerID         string              `json:"askerID" xml:"askerID"`
    AskerNick       string              `json:"askerNick" xml:"askerNick"`
    AskerIP         string              `json:"askerIP" xml:"askerIP"`
    OfferScore      string              `json:"offerScore" xml:"offerScore"`
    PicURL          string              `json:"picURL" xml:"picURL"`
    AnswerCnt       string              `json:"answerCnt" xml:"answerCnt"`
    ClickCnt        string              `json:"clickCnt" xml:"clickCnt"`
    BeginTime       string              `json:"beginTime" xml:"beginTime"`
    EndTime         string              `json:"endTime" xml:"endTime"`
    PollingTime     string              `json:"pollingTime" xml:"pollingTime"`
    BroadcastTime   string              `json:"broadcastTime" xml:"broadcastTime"`
    OfferTime       string              `json:"offerTime" xml:"offerTime"`
    BaseScore       string              `json:"baseScore,omitempty" xml:"baseScore,omitempty"`
    FinalScore      string              `json:"finalScore,omitempty" xml:"finalScore,omitempty"`
    MineScore       string              `json:"minescore" xml:"minescore"`
    Supplements     string              `json:"supplements" xml:"supplements"`
    Comments        string              `json:"comments" xml:"comments"`
    BestAnswers     []TAnswer           `json:"bestAnswers" xml:"bestAnswers>answer"`
    PKAnswers       []TAnswer           `json:"pkAnswers" xml:"pkAnswers>answer"`
    OtherAnswers    []TAnswer           `json:"otherAnswers" xml:"otherAnswers>answer"`
}

func RemoveHtml(s string) string {
    //将HTML标签全转换成小写
    src := html.UnescapeString(s)
    re, _ := regexp.Compile("\\<[\\S\\s]+?\\>")
    src = re.ReplaceAllStringFunc(src, strings.ToLower)

    //去除STYLE
    re, _ = regexp.Compile("\\<style[\\S\\s]+?\\</style\\>")
    src = re.ReplaceAllString(src, "")

    //去除SCRIPT
    re, _ = regexp.Compile("\\<script[\\S\\s]+?\\</script\\>")
    src = re.ReplaceAllString(src, "")

    //去除所有尖括号内的HTML代码，并换成换行符
    re, _ = regexp.Compile("\\<[\\S\\s]+?\\>")
    src = re.ReplaceAllString(src, "\n")

    //去除连续的换行符
    re, _ = regexp.Compile("\\s{2,}")
    src = re.ReplaceAllString(src, "\n")
    src = strings.Replace(src, "\n", " ", -1)
    src = strings.Replace(src, "&deg;", "°", -1)
    return src
}

func XML2Json(xmlstr string) (jsonstr string, err error) {
    var question TQuestion
    err = xml.Unmarshal([] byte(xmlstr), &question)
    if err != nil {
        return
    }
    jsonbytes, err := json.Marshal(question)
    if err != nil {
        return
    }
    jsonstr = string(jsonbytes)
    return
}

func XML2Question(xmlstr string) (question TQuestion, err error) {
    err = xml.Unmarshal([] byte(xmlstr), &question)
    return
}

func main_() {
    fname := os.Args[1]
    file, err := os.Open(fname)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Buffer([]byte{}, bufio.MaxScanTokenSize*10)
    //tokens := make(chan struct {}, 32)
    wg := new(sync.WaitGroup)
    for scanner.Scan() {
        items := strings.Split(scanner.Text(), "\t")
        idstr, mergeTime, xmltext := "", "", ""
        if len(items) == 3 {
            idstr, mergeTime, xmltext = items[0], items[1], items[2]
        } else {
            continue
        }
        _ = idstr
        _ = mergeTime
        jsonstr, err := XML2Json(xmltext)
        if err != nil {
            log.Printf("%s\n", err)
            continue
        }
        log.Printf(jsonstr)
    }
    fmt.Printf("%#v\n", "completed")
    wg.Wait()
    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
}
