# IMY 聊天应用前端

## 功能特性

### 1. 用户认证
- 邮箱密码登录
- 邮箱密码注册
- 邮箱验证码注册
- 自动保存登录状态

### 2. 好友管理
- 搜索用户
- 发送好友请求
- 处理好友验证（接受/拒绝）
- 查看好友列表
- 实时好友请求通知

### 3. 聊天功能
- 创建私聊会话
- 创建群聊会话
- 发送文本消息
- 查看消息历史
- 实时消息更新
- 会话列表管理

### 4. 用户界面
- 响应式设计
- 现代化UI界面
- 实时状态更新
- 友好的错误提示

## 快速开始

### 1. 启动后端服务
确保后端服务运行在 `http://localhost:8081`

### 2. 启动前端服务

#### 方法1：使用Python HTTP服务器
```bash
cd frontend
python3 -m http.server 8080
```

#### 方法2：使用Node.js http-server
```bash
npm install -g http-server
cd frontend
http-server -p 8080
```

#### 方法3：直接打开文件
由于使用了CORS，建议使用HTTP服务器方式。

### 3. 访问应用
打开浏览器访问 `http://localhost:8080`

## 使用说明

### 注册新用户
1. 点击"立即注册"
2. 输入邮箱、密码
3. 点击"获取验证码"获取邮箱验证码
4. 输入验证码
5. 点击"注册"

### 登录
1. 输入注册时的邮箱和密码
2. 点击"登录"
3. 登录成功后自动跳转到主界面

### 添加好友
1. 切换到"好友"标签页
2. 在搜索框输入好友邮箱
3. 点击搜索结果添加好友
4. 等待对方接受好友请求

### 开始聊天
1. 在好友列表点击"聊天"按钮
2. 或在聊天列表选择已有会话
3. 在输入框输入消息按回车发送

### 处理好友请求
1. 好友标签页有红点表示有新请求
2. 点击好友请求查看详情
3. 选择接受或拒绝

## 技术栈

- **前端框架**: 原生HTML5 + JavaScript
- **样式框架**: TailwindCSS
- **图标库**: FontAwesome 6
- **HTTP客户端**: Fetch API

## API端点

### 认证相关
- POST `/api/auth/emailPasswordLogin` - 邮箱密码登录
- POST `/api/auth/emailPasswordRegister` - 邮箱密码注册
- POST `/api/auth/getEmailCode` - 获取邮箱验证码

### 好友管理
- POST `/api/friend/searchUser` - 搜索用户
- POST `/api/friend/addFriend` - 添加好友
- POST `/api/friend/getFriendList` - 获取好友列表
- POST `/api/friend/validFriendList` - 获取好友验证列表
- POST `/api/friend/validFriend` - 处理好友验证

### 聊天功能
- POST `/api/chat/createPrivateConversation` - 创建私聊
- POST `/api/chat/getConversations` - 获取会话列表
- POST `/api/chat/getMessages` - 获取消息历史
- POST `/api/chat/sendMessage` - 发送消息

## 开发说明

### 项目结构
```
frontend/
├── index.html          # 主页面
├── app.js             # 前端逻辑
├── README.md          # 说明文档
└── assets/            # 静态资源（可选）
```

### 配置修改
在 `app.js` 中修改 `API_BASE_URL` 以适应不同的后端地址：

```javascript
const API_BASE_URL = 'http://your-backend-address:port';
```

### 功能扩展
- 添加文件上传功能
- 实现语音消息
- 添加表情包支持
- 实现消息已读状态
- 添加群聊管理功能

## 注意事项

1. **跨域问题**: 确保后端服务已正确配置CORS
2. **Token存储**: 使用localStorage存储登录token
3. **自动刷新**: 每30秒自动刷新会话和好友列表
4. **错误处理**: 所有API调用都有错误处理和用户提示

## 故障排除

### 登录失败
- 检查后端服务是否运行
- 确认邮箱和密码正确
- 查看浏览器控制台错误信息

### 无法添加好友
- 确认对方邮箱存在
- 检查网络连接
- 查看后端日志

### 消息发送失败
- 确认已选择会话
- 检查网络连接
- 查看token是否过期