package getbytid

import (
	"github.com/astaxie/beego"
)

func ConfigRoutes() {
	beego.Router("/getbytid", &GetByTidController{})
}
