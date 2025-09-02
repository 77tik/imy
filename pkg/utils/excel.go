package utils

import (
	"github.com/xuri/excelize/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type StyleOption func(f *excelize.File) error

func CreateExcelContent(titles []any, content [][]any, styleOpts ...StyleOption) (*excelize.File, error) {
	// 创建excel
	// 会初始化一个默认的sheet，如果再newsheet的话，会往后加sheet，这倒是没什么，但是上传excel文件的时候如果默认读取第一个sheet就遭老罪了，会什么都读不到
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			logx.Error(err)
		}
	}()

	index, err := f.NewSheet("Sheet1")
	if err != nil {
		return nil, err
	}

	// 设置默认工作表
	f.SetActiveSheet(index)

	// 设置表头
	for i, title := range titles {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return nil, err
		}
		err = f.SetCellValue("Sheet1", cell, title)
		if err != nil {
			return nil, err
		}
	}

	// 保存数据
	for i, values := range content {
		for j, value := range values {
			cell, err := excelize.CoordinatesToCellName(j+1, i+2)
			if err != nil {
				return nil, err
			}
			err = f.SetCellValue("Sheet1", cell, value)
			if err != nil {
				return nil, err
			}
		}
	}

	for _, opt := range styleOpts {
		err = opt(f)
		logx.Error(err)
		return nil, err
	}

	return f, nil
}
