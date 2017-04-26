package http

import (
	"github.com/astaxie/beego"
	"github.com/lokicui/mlt/http/home"
	"github.com/lokicui/mlt/http/morelikethis"
)

func Start(addr string) {
	home.ConfigRoutes()
    morelikethis.ConfigRoutes()
	beego.Run(addr)
}
