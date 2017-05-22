package home

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/lokicui/mlt/http/morelikethis"
)

type HomeController struct {
	beego.Controller
}

func (this *HomeController) Get() {
    request, retcode, err := morelikethis.GenMltRequest(this.Input())
    if err != nil {
        logs.Debug(err, retcode)
    } else {
        result, keywordsMap := morelikethis.GetMoreLikeThisResult(request)
        this.Data["L"] = result
        this.Data["Keywords"] = keywordsMap
    }
    //logs.Debug("req=", request)
    this.Data["Form"] = request
	this.TplName = "home/index.html"
}
