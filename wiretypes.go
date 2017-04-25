package main

type MltRequest struct {
    Qid         int         //相关问题/pk/帖子/投票的id
    Query       string      //相关问题的标题
    UUID        string      //uuid
    Preference  string      //请求路由值
    Start       int
    Limit       int
    Typev       int         //需要什么类型的返回数据 1:问题 2
    Tids        []int
    Pretty      bool
}

func NewMltRequest() *MltRequest {
    self := &MltRequest{
        Qid:        0,
        Query:      "",
        UUID:       "default_uuid",
        Preference: "default_preference",
        Start:      0,
        Limit:      15,
        Typev:      0,
        Tids:       []int{},
        Pretty:     false,
    }
    return self
}
