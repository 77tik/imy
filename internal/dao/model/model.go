package model

import (
	"imy/internal/types"
)

func (i *Verify) DTO() types.VerifyInfo {
	return types.VerifyInfo{
		Id:     i.ID,
		SendId: i.SendID,
		RevId:  i.RevID,
		Status: i.Status,
	}
}
