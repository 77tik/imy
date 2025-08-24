// API配置
const API_BASE_URL = 'http://localhost:8081';
let currentUser = null;
let token = null;
let currentConversationId = null;

// 工具函数
function showToast(message, type = 'success') {
    const toast = document.createElement('div');
    toast.className = `fixed top-4 right-4 px-4 py-2 rounded-lg text-white ${type === 'success' ? 'bg-green-500' : 'bg-red-500'}`;
    toast.textContent = message;
    document.body.appendChild(toast);
    setTimeout(() => toast.remove(), 3000);
}

function showLoading() {
    const loading = document.createElement('div');
    loading.id = 'loading';
    loading.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
    loading.innerHTML = '<div class="text-white text-xl"><i class="fas fa-spinner fa-spin mr-2"></i>加载中...</div>';
    document.body.appendChild(loading);
}

function hideLoading() {
    const loading = document.getElementById('loading');
    if (loading) loading.remove();
}

// API调用函数
async function apiCall(endpoint, data = {}, method = 'POST') {
    const headers = {
        'Content-Type': 'application/json',
    };
    
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }
    
    const options = {
        method,
        headers,
    };
    
    if (method !== 'GET' && data) {
        options.body = JSON.stringify(data);
    }
    
    try {
        const response = await fetch(`${API_BASE_URL}${endpoint}`, options);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        return await response.json();
    } catch (error) {
        console.error('API调用失败:', error);
        throw error;
    }
}

// 认证相关
function showLogin() {
    document.getElementById('loginForm').style.display = 'block';
    document.getElementById('registerForm').style.display = 'none';
}

function showRegister() {
    document.getElementById('loginForm').style.display = 'none';
    document.getElementById('registerForm').style.display = 'block';
}

async function login() {
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    
    if (!email || !password) {
        showToast('请填写邮箱和密码', 'error');
        return;
    }
    
    showLoading();
    try {
        const response = await fetch(`${API_BASE_URL}/api/auth/emailPasswordLogin`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        });
        
        if (response.ok) {
            const data = await response.json();
            token = response.headers.get('Authorization')?.replace('Bearer ', '') || '';
            if (data.uuid) {
                currentUser = { uuid: data.uuid, email };
                localStorage.setItem('token', token);
                localStorage.setItem('userEmail', email);
                showApp();
                showToast('登录成功');
            }
        } else {
            showToast('登录失败，请检查邮箱和密码', 'error');
        }
    } catch (error) {
        showToast('登录失败，请稍后重试', 'error');
    } finally {
        hideLoading();
    }
}

async function register() {
    const email = document.getElementById('regEmail').value;
    const password = document.getElementById('regPassword').value;
    const code = document.getElementById('code').value;
    
    if (!email || !password || !code) {
        showToast('请填写所有字段', 'error');
        return;
    }
    
    showLoading();
    try {
        const response = await fetch(`${API_BASE_URL}/api/auth/emailPasswordRegister`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password, code })
        });
        
        if (response.ok) {
            showToast('注册成功，请登录');
            showLogin();
        } else {
            showToast('注册失败，请检查信息', 'error');
        }
    } catch (error) {
        showToast('注册失败，请稍后重试', 'error');
    } finally {
        hideLoading();
    }
}

async function getEmailCode() {
    const email = document.getElementById('regEmail').value;
    if (!email) {
        showToast('请先输入邮箱', 'error');
        return;
    }
    
    const btn = document.getElementById('getCodeBtn');
    btn.disabled = true;
    btn.textContent = '发送中...';
    
    try {
        const response = await fetch(`${API_BASE_URL}/api/auth/getEmailCode`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email })
        });
        
        if (response.ok) {
            showToast('验证码已发送到邮箱');
            let countdown = 60;
            const timer = setInterval(() => {
                countdown--;
                btn.textContent = `${countdown}秒后重试`;
                if (countdown <= 0) {
                    clearInterval(timer);
                    btn.disabled = false;
                    btn.textContent = '获取验证码';
                }
            }, 1000);
        } else {
            showToast('发送失败，请稍后重试', 'error');
            btn.disabled = false;
            btn.textContent = '获取验证码';
        }
    } catch (error) {
        showToast('发送失败，请稍后重试', 'error');
        btn.disabled = false;
        btn.textContent = '获取验证码';
    }
}

// 应用主界面
function showApp() {
    document.getElementById('loginPage').style.display = 'none';
    document.getElementById('app').style.display = 'block';
    
    document.getElementById('userEmail').textContent = localStorage.getItem('userEmail') || '用户';
    
    loadConversations();
    loadFriends();
    loadFriendRequests();
}

function logout() {
    localStorage.removeItem('token');
    localStorage.removeItem('userEmail');
    token = null;
    currentUser = null;
    document.getElementById('loginPage').style.display = 'flex';
    document.getElementById('app').style.display = 'none';
}

// 聊天功能
function showChat() {
    document.getElementById('chatTab').className = 'flex-1 py-3 text-center font-medium text-blue-600 border-b-2 border-blue-600';
    document.getElementById('friendsTab').className = 'flex-1 py-3 text-center font-medium text-gray-600';
    document.getElementById('chatList').style.display = 'block';
    document.getElementById('friendsList').style.display = 'none';
}

function showFriends() {
    document.getElementById('chatTab').className = 'flex-1 py-3 text-center font-medium text-gray-600';
    document.getElementById('friendsTab').className = 'flex-1 py-3 text-center font-medium text-blue-600 border-b-2 border-blue-600';
    document.getElementById('chatList').style.display = 'none';
    document.getElementById('friendsList').style.display = 'block';
}

async function loadConversations() {
    try {
        const response = await apiCall('/api/chat/getConversations', { uuid: currentUser.uuid });
        const conversations = response.conversations || [];
        
        const container = document.getElementById('conversations');
        container.innerHTML = '';
        
        conversations.forEach(conv => {
            const div = document.createElement('div');
            div.className = 'p-4 hover:bg-gray-50 cursor-pointer';
            div.onclick = () => openConversation(conv);
            div.innerHTML = `
                <div class="flex items-center">
                    <div class="w-12 h-12 bg-blue-500 rounded-full flex items-center justify-center text-white font-bold">
                        ${conv.name.charAt(0).toUpperCase()}
                    </div>
                    <div class="ml-3 flex-1">
                        <div class="font-semibold">${conv.name}</div>
                        <div class="text-sm text-gray-500">${conv.memberCount} 人</div>
                    </div>
                </div>
            `;
            container.appendChild(div);
        });
    } catch (error) {
        console.error('加载会话失败:', error);
    }
}

async function openConversation(conv) {
    currentConversationId = conv.conversationId;
    document.getElementById('chatTitle').textContent = conv.name;
    document.getElementById('chatSubtitle').textContent = `${conv.memberCount} 人`;
    document.getElementById('messageArea').classList.remove('hidden');
    document.getElementById('inputArea').classList.remove('hidden');
    
    loadMessages(conv.conversationId);
}

async function loadMessages(conversationId) {
    try {
        const response = await apiCall('/api/chat/getMessages', {
            uuid: currentUser.uuid,
            conversationId: conversationId,
            limit: 50
        });
        
        const messages = response.messages || [];
        const container = document.getElementById('messages');
        container.innerHTML = '';
        
        messages.reverse().forEach(msg => {
            const div = document.createElement('div');
            div.className = `flex ${msg.sendUuid === currentUser.uuid ? 'justify-end' : 'justify-start'}`;
            div.innerHTML = `
                <div class="max-w-xs lg:max-w-md">
                    <div class="${msg.sendUuid === currentUser.uuid ? 'bg-blue-500 text-white' : 'bg-gray-200'} rounded-lg px-4 py-2">
                        <div class="text-sm">${msg.content}</div>
                        <div class="text-xs mt-1 ${msg.sendUuid === currentUser.uuid ? 'text-blue-100' : 'text-gray-500'}">
                            ${new Date(msg.createdAt).toLocaleTimeString()}
                        </div>
                    </div>
                </div>
            `;
            container.appendChild(div);
        });
        
        container.scrollTop = container.scrollHeight;
    } catch (error) {
        console.error('加载消息失败:', error);
    }
}

async function sendMessage() {
    const input = document.getElementById('messageInput');
    const content = input.value.trim();
    
    if (!content || !currentConversationId) return;
    
    try {
        const response = await apiCall('/api/chat/sendMessage', {
            uuid: currentUser.uuid,
            conversationId: currentConversationId,
            clientMsgId: Date.now().toString(),
            msgType: 1,
            content: content
        });
        
        input.value = '';
        loadMessages(currentConversationId);
    } catch (error) {
        showToast('发送消息失败', 'error');
    }
}

// 好友功能
async function loadFriends() {
    try {
        const response = await apiCall('/api/friend/getFriendList', { uuid: currentUser.uuid });
        const friends = response.friends || [];
        
        const container = document.getElementById('friends');
        container.innerHTML = '';
        
        friends.forEach(friend => {
            const div = document.createElement('div');
            div.className = 'flex items-center p-3 bg-gray-50 rounded-lg';
            div.innerHTML = `
                <div class="w-10 h-10 bg-green-500 rounded-full flex items-center justify-center text-white font-bold">
                    ${friend.notice.charAt(0).toUpperCase()}
                </div>
                <div class="ml-3 flex-1">
                    <div class="font-medium">${friend.notice}</div>
                    <div class="text-sm text-gray-500">${friend.uuid}</div>
                </div>
                <button onclick="startChat('${friend.uuid}')" class="px-3 py-1 bg-blue-500 text-white rounded text-sm hover:bg-blue-600">
                    聊天
                </button>
            `;
            container.appendChild(div);
        });
    } catch (error) {
        console.error('加载好友失败:', error);
    }
}

async function startChat(friendUuid) {
    try {
        const response = await apiCall('/api/chat/createPrivateConversation', {
            uuid: currentUser.uuid,
            peerUuid: friendUuid
        });
        
        if (response.conversationId) {
            loadConversations();
            showChat();
            openConversation(response);
        }
    } catch (error) {
        showToast('创建聊天失败', 'error');
    }
}

async function searchUser() {
    const email = document.getElementById('friendSearch').value;
    if (!email) return;
    
    try {
        const response = await apiCall('/api/friend/searchUser', { email });
        if (response.revId) {
            document.getElementById('friendEmail').value = email;
            showAddFriendModal();
        } else {
            showToast('未找到用户', 'error');
        }
    } catch (error) {
        showToast('搜索失败', 'error');
    }
}

async function addFriend() {
    const email = document.getElementById('friendEmail').value;
    if (!email) return;
    
    try {
        const response = await apiCall('/api/friend/addFriend', {
            uuid: currentUser.uuid,
            revId: email
        });
        
        showToast('好友请求已发送');
        closeAddFriendModal();
    } catch (error) {
        showToast('添加好友失败', 'error');
    }
}

async function loadFriendRequests() {
    try {
        const response = await apiCall('/api/friend/validFriendList', { uuid: currentUser.uuid });
        const requests = response.valids || [];
        
        if (requests.length > 0) {
            const indicator = document.createElement('div');
            indicator.className = 'absolute top-2 right-2 w-3 h-3 bg-red-500 rounded-full';
            indicator.id = 'friendRequestIndicator';
            document.querySelector('#friendsTab').appendChild(indicator);
        }
    } catch (error) {
        console.error('加载好友请求失败:', error);
    }
}

// 模态框控制
function showAddFriendModal() {
    document.getElementById('addFriendModal').classList.remove('hidden');
    document.getElementById('addFriendModal').classList.add('flex');
}

function closeAddFriendModal() {
    document.getElementById('addFriendModal').classList.add('hidden');
    document.getElementById('addFriendModal').classList.remove('flex');
    document.getElementById('friendEmail').value = '';
}

function showFriendRequestsModal() {
    document.getElementById('friendRequestsModal').classList.remove('hidden');
    document.getElementById('friendRequestsModal').classList.add('flex');
    loadFriendRequestsList();
}

function closeFriendRequestsModal() {
    document.getElementById('friendRequestsModal').classList.add('hidden');
    document.getElementById('friendRequestsModal').classList.remove('flex');
}

async function loadFriendRequestsList() {
    try {
        const response = await apiCall('/api/friend/validFriendList', { uuid: currentUser.uuid });
        const requests = response.valids || [];
        
        const container = document.getElementById('friendRequests');
        container.innerHTML = '';
        
        requests.forEach(req => {
            const div = document.createElement('div');
            div.className = 'flex items-center justify-between p-3 bg-gray-50 rounded-lg';
            div.innerHTML = `
                <div>
                    <div class="font-medium">${req.revId}</div>
                    <div class="text-sm text-gray-500">状态: ${req.revStatus === 1 ? '待处理' : req.revStatus === 2 ? '已接受' : '已拒绝'}</div>
                </div>
                <div class="space-x-2">
                    <button onclick="handleFriendRequest(${req.id}, 2)" class="px-3 py-1 bg-green-500 text-white rounded text-sm hover:bg-green-600">接受</button>
                    <button onclick="handleFriendRequest(${req.id}, 3)" class="px-3 py-1 bg-red-500 text-white rounded text-sm hover:bg-red-600">拒绝</button>
                </div>
            `;
            container.appendChild(div);
        });
    } catch (error) {
        console.error('加载好友请求列表失败:', error);
    }
}

async function handleFriendRequest(verifyId, status) {
    try {
        await apiCall('/api/friend/validFriend', {
            uuid: currentUser.uuid,
            verifyId: verifyId,
            status: status
        });
        
        showToast(status === 2 ? '已接受好友请求' : '已拒绝好友请求');
        loadFriends();
        loadFriendRequestsList();
    } catch (error) {
        showToast('处理好友请求失败', 'error');
    }
}

// 事件监听器
document.getElementById('friendSearch').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') {
        searchUser();
    }
});

// 初始化
document.addEventListener('DOMContentLoaded', function() {
    const savedToken = localStorage.getItem('token');
    const savedEmail = localStorage.getItem('userEmail');
    
    if (savedToken && savedEmail) {
        token = savedToken;
        currentUser = { uuid: savedEmail }; // 这里应该解析token获取真实UUID
        showApp();
    }
});

// 每30秒刷新一次会话和好友列表
setInterval(() => {
    if (currentUser) {
        loadConversations();
        loadFriends();
        loadFriendRequests();
    }
}, 30000);