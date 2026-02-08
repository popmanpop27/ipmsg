package models


type IPmsgRequest struct {
	From  string
	Len   int 
	Date  int64
	Msg   string
	Alias string
}