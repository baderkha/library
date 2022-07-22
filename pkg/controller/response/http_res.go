package response

type HTTPResponse[t any] struct {
	Data          t      `json:"data"`
	ErrorMessage  string `json:"error_message"`
	ServerMessage string `json:"server_message"`
}

func New[t any](item t) *HTTPResponse[t] {
	return &HTTPResponse[t]{
		Data:          item,
		ErrorMessage:  "",
		ServerMessage: "ok",
	}
}

func NewError(err error) *HTTPResponse[any] {
	var errMessage string = "UNKOWN ERROR REASON ..."
	if err != nil {
		errMessage = err.Error()
	}
	return &HTTPResponse[any]{
		Data:          nil,
		ErrorMessage:  errMessage,
		ServerMessage: "error",
	}
}
