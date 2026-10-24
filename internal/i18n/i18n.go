package i18n

func Lang(msg, lang string) string {
	if lang == "zh_CN" {
		switch msg {
		case "invalidToke":
			return "无效的token"
		case "timeout":
			return "会话超时，请重新登入！"
		case "freeUser":
			return "免费用户"
		case "invalidParms":
			return "无效的参数"
		case "alreadySignIn1":
			return "今日已领取"
		case "freeGetTime":
			return "领取免费时长"
		case "alreadySignIn2":
			return "您今天已经领取过哦！"
		case "getTimeOK":
			return "领取成功，您获得了30分钟"
		case "normal":
			return "正常"
		case "expireAccount":
			return "您的账号已过期"
		case "serverException":
			return "服务器发送异常，请联系客服或稍后再试"
		case "resetPwdOK":
			return "重置密码成功，请重新登录客户端"
		case "resetFailed":
			return "修改失败"
		case "invalidAccount":
			return "用户邮箱或密码错误,或者账号未激活"
		case "testVersionMSG":
			return "测试版仅限VIP且有可用天数的用户使用！"
		case "errorFormatEmail":
			return "请输入正确邮箱格式"
		case "inputPwd":
			return "请输入密码"
		case "existUserEmail":
			return "用户邮箱已注册"
		case "sendEmail":
			return "我们已经发送一封验证邮件到%s\\n,请点击里面的链接验证您的邮箱！"
		case "registerOK":
			return "您已注册成功，但是在为您发送激活邮件时系统发生了错误,请联系网站管理员以获得帮助"
		}
	}
	switch msg {
	case "invalidToke":
		return "Invalid token"
	case "timeout":
		return "The session is time out, please login in again!"
	case "freeUser":
		return "Free User"
	case "invalidParms":
		return "Invalid parameters"
	case "alreadySignIn1":
		return "Today has been received"
	case "freeGetTime":
		return "Receive free time"
	case "alreadySignIn2":
		return "You've got it today!"
	case "getTimeOK":
		return "You got 2 hours"
	case "normal":
		return "Normal"
	case "expireAccount":
		return "Your account is expired"
	case "serverException":
		return "Server occur error, please try again!"
	case "resetPwdOK":
		return "Reset password successful, plase re-login in."
	case "resetFailed":
		return "Reset failed"
	case "invalidAccount":
		return "The uese email or password error, or tha account is not active"
	case "testVersionMSG":
		return "Beta version is only support for VIP user and have limit days!"
	case "errorFormatEmail":
		return "Please input right Email!"
	case "inputPwd":
		return "Please input password!"
	case "existUserEmail":
		return "The email is exist!"
	case "sendEmail":
		return "We send a email to your mailbox %s\\n,please click the link in the email to active your account!"
	case "registerOK":
		return "Your email register successful, but sent active email failed, please contract us to get help, thanks!"
	}
	return msg
}
