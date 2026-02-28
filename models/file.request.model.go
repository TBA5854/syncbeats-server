package models

type FileUploadRequestModel struct {
	UserId   string `json:"user_id" form:"user_id"`
	FileName string `json:"file_name" form:"file_name"`
	Hash     string `json:"hash" form:"hash"` // client-computed MD5 hex; verified server-side
}

type FileDownloadRequestModel struct {
	UserId string `json:"user_id"`
	FileId string `query:"file_id"`
}
