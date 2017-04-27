package http

import (
	"github.com/astaxie/beego"
	"github.com/lokicui/mlt/http/home"
	"github.com/lokicui/mlt/http/morelikethis"
	"github.com/lokicui/mlt/http/getbytid"
)

func Start(addr string) {
	home.ConfigRoutes()
    morelikethis.ConfigRoutes()
    getbytid.ConfigRoutes()
	beego.Run(addr)
}
