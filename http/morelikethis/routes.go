package morelikethis

import (
    "github.com/astaxie/beego"
    "github.com/lokicui/mlt/http/morelikethis/taginfo"
    "strconv"
)

func GetTagNameById(idstr string) string {
    id, err := strconv.Atoi(idstr)
    if err != nil {
        return idstr
    }
    if info, ok := taginfo.GetTagInfoById(id); ok {
        return info.Name
    }
    return idstr
}

func ConfigRoutes() {
    taginfo.Init("conf/db_taginfo.txt")
    beego.AddFuncMap("id2name", GetTagNameById)
    beego.Handler("/more_like_this", MakeHandler(moreLikeThisHandler))
}
