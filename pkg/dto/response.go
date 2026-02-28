package dto

type (
	Response struct {
		Success bool `json:"success"`
		Data    any  `json:"data"`
	}

	ErrorData struct {
		Message string `json:"message"`
	}

	LoginResponseData struct {
		Message      string `json:"message"`
		Token        string `json:"token"`
		GitLabUserID int    `json:"gitlab_user_id"`
	}
)

func SuccessResponse(data any) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

func ErrorResponse(message string) Response {
	return Response{
		Success: false,
		Data:    ErrorData{Message: message},
	}
}
