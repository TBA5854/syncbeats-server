package models

type FileUploadResponseModel struct {
	FileId string `json:"file_id"`
}

type FileListItemModel struct {
	FileId   string `json:"file_id"`
	FileName string `json:"file_name"`
}

type FileListResponseModel struct {
	Files []FileListItemModel `json:"files"`
}
