package models

import "fmt"

type IPResponse struct {
	Succes bool
	Error *string
}

func (ier *IPResponse) DecodeToString() string {
	res := fmt.Sprintf("ipmsg\nsucces:%v\n", ier.Succes)

	if ier.Error != nil {
		res += fmt.Sprintf("error:%s", *ier.Error)
	}

	return res
}