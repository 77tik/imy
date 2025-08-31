package errcode

import (
	"imy/pkg/utils"
)

var (
	ErrInternalErr    = utils.NewBaseError(1000, "服务器内部错误")
	ErrUnknownErr     = utils.NewBaseError(1001, "未知错误")
	ErrInvalidParam   = utils.NewBaseError(1002, "参数错误")
	ErrDataCreateFail = utils.NewBaseError(1003, "数据添加失败")
	ErrDataDeleteFail = utils.NewBaseError(1004, "数据删除失败")
	ErrDataModifyFail = utils.NewBaseError(1005, "数据修改失败")
	ErrDataQueryFail  = utils.NewBaseError(1006, "数据查询失败")

	ErrAuth                  = utils.NewBaseError(1100, "认证错误")
	ErrAuthInvalidParam      = utils.NewBaseError(1101, "用户名或者密码错误")
	ErrAuthCaptchaType       = utils.NewBaseError(1102, "验证码类型错误")
	ErrAuthCaptchaCreate     = utils.NewBaseError(1103, "验证码生成失败")
	ErrAuthCaptchaError      = utils.NewBaseError(1104, "验证码错误")
	ErrAuthDecryptFail       = utils.NewBaseError(1105, "验证失败")
	ErrAuthUserNotFund       = utils.NewBaseError(1106, "用户不存在")
	ErrAuthUserExist         = utils.NewBaseError(1107, "用户已存在")
	ErrAuthTokenExpire       = utils.NewBaseError(1108, "token过期")
	ErrAuthTokenFailed       = utils.NewBaseError(1109, "token解析失败")
	ErrAuthTokenNil          = utils.NewBaseError(1110, "token为空")
	ErrAuthTokenUseless      = utils.NewBaseError(1111, "token无效")
	ErrAuthSession           = utils.NewBaseError(1112, "会话信息出错")
	ErrJsonMarshal           = utils.NewBaseError(1113, "序列化json失败")
	ErrAuthTokenCreateFailed = utils.NewBaseError(1114, "token生成失败")
	ErrPasswordGenerate      = utils.NewBaseError(1115, "密码哈希失败")

	ErrTime         = utils.NewBaseError(1201, "时间解析错误")
	ErrFileNotFund  = utils.NewBaseError(1202, "文件不存在")
	ErrFileOpenFail = utils.NewBaseError(1203, "文件读取失败")
	ErrExcel        = utils.NewBaseError(1204, "Excel创建失败")
	ErrFileSave     = utils.NewBaseError(1205, "文件保存失败")

	ErrAlgorithm = utils.NewBaseError(1301, "算法计算错误")
	ErrRedisSet  = utils.NewBaseError(1302, "redis存储key失败")

	ErrFriendAlreadyExist = utils.NewBaseError(1401, "好友已存在")
	ErrVerifyCreate       = utils.NewBaseError(1402, "创建验证失败")
	ErrVerifyDeal         = utils.NewBaseError(1403, "处理验证失败")
	ErrVerifyNotFound     = utils.NewBaseError(1404, "该好友验证不存在")
	ErrVerifyExist        = utils.NewBaseError(1405, "该条验证已经存在")
)
