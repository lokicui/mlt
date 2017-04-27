package getbytid

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/lokicui/mlt/http/morelikethis"
)

type GetByTidController struct {
	beego.Controller
}

func (this *GetByTidController) Get() {
    request, retcode, err := morelikethis.GenGetByTidRequest(this.Input())
    if err != nil {
        logs.Debug(err, retcode)
    } else {
        result := morelikethis.GetByTidResult(request)
        this.Data["L"] = result
    }
    //logs.Debug("req=", request)
    this.Data["Form"] = request
	this.TplName = "getbytid/index.html"
}
