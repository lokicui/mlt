package morelikethis

type MltRequest struct {
    Qid         int         `form:"-"` //相关问题/pk/帖子/投票的id
    Query       string      `form:"query"`//相关问题的标题
    UUID        string      `form:"-"`//uuid
    Preference  string      `form:"-"`//请求路由值
    Start       int         `form:"-"`
    Limit       int         `form:"-"`
    Type        int         `form:"type"`//需要什么类型的返回数据 1:问题 2
    Tids        string      `from:"tids"`
    TidsList     []int       `form:"-"`
    Pretty      bool        `form:"-"`
    Debug       bool        `form:"debug"`
}

func NewMltRequest() *MltRequest {
    self := &MltRequest{
        Qid:        0,
        Query:      "",
        UUID:       "default_uuid",
        Preference: "default_preference",
        Start:      0,
        Limit:      15,
        Type:      0,
        Tids:       "",
        TidsList:   []int{},
        Pretty:     false,
        Debug:      false,
    }
    return self
}

type GetByTidRequest struct {
    Tid         string      `form:"tid"` //tagid
    UUID        string      `form:"-"`//uuid
    Preference  string      `form:"-"`//请求路由值
    Start       int         `form:"-"`
    Limit       int         `form:"-"`
    Type        int         `form:"type"`//需要什么类型的返回数据 1:问题 2
    Pretty      bool        `form:"-"`
    Debug       bool        `form:"debug"`
}
func NewGetByTidRequest() * GetByTidRequest {
    self := &GetByTidRequest {
        Tid: "",
        UUID:       "default_uuid",
        Preference: "default_preference",
        Start:      0,
        Limit:      15,
        Type:      0,
        Pretty:     false,
        Debug:      false,
    }
    return self
}
