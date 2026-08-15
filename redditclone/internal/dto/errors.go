package dto

type ValidationError struct {
	Location string `json:"location"`
	Param    string `json:"param"`
	Value    string `json:"value"`
	Msg      string `json:"msg"`
}

type ErrorResponse struct {
	Errors []ValidationError `json:"errors"`
}
