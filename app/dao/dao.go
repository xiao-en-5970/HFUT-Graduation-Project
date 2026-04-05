package dao

func User() *UserStore                         { return &UserStore{} }
func UserCert() *UserCertStore                 { return &UserCertStore{} }
func Article() *ArticleStore                   { return &ArticleStore{} }
func School() *SchoolStore                     { return &SchoolStore{} }
func Comment() *CommentStore                   { return &CommentStore{} }
func Collect() *CollectStore                   { return &CollectStore{} }
func CollectItem() *CollectItemStore           { return &CollectItemStore{} }
func Like() *LikeStore                         { return &LikeStore{} }
func Good() *GoodStore                         { return &GoodStore{} }
func Order() *OrderStore                       { return &OrderStore{} }
func OrderMessage() *OrderMessageStore         { return &OrderMessageStore{} }
func OrderMessageRead() *OrderMessageReadStore { return &OrderMessageReadStore{} }
func UserLocation() *UserLocationStore         { return &UserLocationStore{} }
