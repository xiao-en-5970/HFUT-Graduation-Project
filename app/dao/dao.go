package dao

func User() *UserStore               { return &UserStore{} }
func Article() *ArticleStore         { return &ArticleStore{} }
func Comment() *CommentStore         { return &CommentStore{} }
func Collect() *CollectStore         { return &CollectStore{} }
func CollectItem() *CollectItemStore { return &CollectItemStore{} }
func Like() *LikeStore               { return &LikeStore{} }
