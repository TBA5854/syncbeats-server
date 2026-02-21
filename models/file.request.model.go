package models

type FileUploadRequestModel struct {
	UserId string	`json:"user_id"`
	FileName string  `json:"file_name"`
}

type FileDownloadRequestModel struct {
	UserId string	`json:"user_id"`
	FileId string  `json:"file_id"`
}	