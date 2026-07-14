package e

const (
	Success      = 200
	InvalidParam = 400
	Unauthorized = 401
	Forbidden    = 403
	NotFound     = 404
	ServerError  = 500
)

var Messages = map[int]string{
	Success:      "操作成功",
	InvalidParam: "参数错误",
	Unauthorized: "未授权",
	Forbidden:    "无权限",
	NotFound:     "资源不存在",
	ServerError:   "系统错误",
}

func Message(code int) string {
	if msg, ok := Messages[code]; ok {
		return msg
	}
	return Messages[ServerError]
}
