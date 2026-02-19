package service

// 各模块 Service 工厂函数，统一入口

func User() *userService       { return &userService{} }
func Article() *articleService { return &articleService{} }
func Comment() *commentService { return &commentService{} }
func Collect() *collectService { return &collectService{} }
func Like() *likeService       { return &likeService{} }
