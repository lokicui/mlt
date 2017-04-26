package morelikethis

import (
    "github.com/astaxie/beego"
)

func ConfigRoutes() {
    beego.Handler("/more_like_this", MakeHandler(moreLikeThisHandler))
}
