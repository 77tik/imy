# 字典表
drop table if exists user;
create table if not exists user
(
    id         int unsigned primary key auto_increment comment '主键id',
    uuid varchar(64) not null default '' comment '用户ID',
    nick_name varchar(128) not null default '' comment '用户昵称',
    password varchar(128) not null default '' comment '加密后的密码',
    email varchar(128) not null default '' comment '邮箱',
    file_name varchar(256) not null default '' comment '头像文件路径',
    abstract varchar(256) not null default '' comment '个性签名',
    phone varchar(11) not null default '' comment '手机号',
    status tinyint not null default 1 comment '1表示正常，2表示禁用',
    gender tinyint not null default 1 comment '1男，2女，3未知',
    last_login_ip varchar(39) not null default '' comment '最后一次登陆的ip地址',
    source tinyint not null default 1 comment '注册来源',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
    ) engine = InnoDB
    auto_increment = 1
    character set = utf8mb4
    collate = utf8mb4_general_ci
    row_format = Dynamic comment ='用户表';

# 好友表
drop table if exists friend;
create table if not exists friend
(
    id         int unsigned primary key auto_increment comment '主键id',
    send_uuid varchar(64) not null default '' comment '发起方',
    rev_uuid varchar(64) not null default '' comment '接收者',
    send_notice varchar(128) not null default '' comment '发起方对对方的备注',
    rev_notice varchar(128) not null default '' comment '接收方对对方的备注',
    source tinyint not null default 1 comment '关系来源比如发送方从群聊中获得，还是搜索获得，默认1代表搜索',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='好友表';

# 好友验证表
drop table if exists friend_verify;
create table if not exists friend_verify
(
    id         int unsigned primary key auto_increment comment '主键id',
    send_uuid varchar(64) not null default '' comment '发起方',
    rev_uuid varchar(64) not null default '' comment '接收者',
    send_status tinyint not null default 1 comment '发送方状态，1表示待处理，2撤销',
    rev_status tinyint not null default 1 comment '接收方状态，1表示未处理，2接受，3拒绝',
    message varchar(128) not null default '' comment '好友验证消息',
    source tinyint not null default 1 comment '关系来源比如发送方从群聊中获得，还是搜索获得，默认1代表搜索',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='好友验证表';



# 聊天会话表
drop table if exists chat_conversation;
create table if not exists chat_conversation
(
    id                 int unsigned primary key auto_increment comment '主键id',
    type               tinyint not null default 1 comment '会话类型，1表示单聊，2表示群聊',
    private_key        varchar(200) not null default '' comment '会话标识',
    create_uuid        varchar(64) not null default '' comment '创建方',
    name               varchar(128) not null default '' comment '会话名称,群聊使用,单聊可以为空',
    member_count       int unsigned not null default 0 comment '成员数',
    last_message_id    bigint unsigned not null default 0 comment '最后一条消息的id，用于展示',
    avatar             varchar(512) not null default '' comment '会话头像（群聊）',
    extra              varchar(1024) not null default '' comment '扩展信息（置顶、公告等）',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',

    unique key uidx_private_key (private_key) using btree,
    index idx_type (type) using btree,
    index idx_create_uuid (create_uuid) using btree,
    index idx_last_message_id (last_message_id) using btree,
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='聊天会话表';



# 聊天会话成员表
drop table if exists chat_conversation_member;
create table if not exists chat_conversation_member
(
    id                   int unsigned primary key auto_increment comment '主键id',
    conversation_id      int unsigned not null default 0 comment '会话id',
    user_uuid            varchar(64) not null default '' comment '成员uuid',
    role                 tinyint not null default 1 comment '成员角色，1表示普通成员，2表示管理员，群聊使用',

    last_read_message_id bigint unsigned not null default 0 comment '最后已读消息ID',
    last_read_at         datetime not null default now() comment '最后已读时间',
    mute_until           datetime not null default '1970-01-01 00:00:00' comment '免打扰截止时间',
    is_pinned            tinyint(1) not null default 0 comment '是否置顶',
    alias                varchar(64) not null default '' comment '群内昵称/备注',
    unread_count         int unsigned not null default 0 comment '未读数（可选缓存字段）',
    extra                varchar(512) not null default '' comment '扩展信息',
    
    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',

    unique key uidx_conv_user (conversation_id, user_uuid) using btree,
    index idx_user_uuid (user_uuid) using btree,
    index idx_conversation_id (conversation_id) using btree,
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='聊天会话成员表';



# 聊天消息表
drop table if exists chat_message;
create table if not exists chat_message
(
    id                    bigint unsigned primary key auto_increment comment '主键id',
    conversation_id       int unsigned not null default 0 comment '会话id',
    send_uuid             varchar(64) not null default '' comment '发送方uuid',
    client_msg_id         varchar(64) not null default '' comment '客户端幂等ID',
    msg_type              tinyint not null default 1 comment '消息类型：1文本、2图片、3语音、4视频、5文件、6系统',
    content               text not null comment '消息内容（文本或JSON）',
    content_extra         varchar(1024) not null default '' comment '内容扩展（图片宽高、文件大小等）',
    reply_to_message_id   bigint unsigned not null default 0 comment '引用/回复的消息ID',
    mentioned_uuids       varchar(1024) not null default '' comment '@用户uuid列表（逗号分隔或JSON）',
    is_system             tinyint(1) not null default 0 comment '是否系统消息',
    is_revoked            tinyint(1) not null default 0 comment '是否撤回',
    revoked_at            datetime null comment '撤回时间',
    
    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',

    unique key uidx_conv_sender_client (conversation_id, send_uuid, client_msg_id) using btree,
    index idx_conv (conversation_id) using btree,
    index idx_conv_id (conversation_id, id) using btree,
    index idx_send_uuid (send_uuid) using btree,
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='聊天信息表';