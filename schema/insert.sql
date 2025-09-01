# 认证表
drop table if exists auth;
create table if not exists auth
(
    id         int unsigned primary key auto_increment comment '主键id',
    account varchar(64) not null default '' comment '用户账号',
    nick_name varchar(64) not null default '' comment '用户昵称',
    password varchar(128) not null default '' comment '加密后的密码',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
    ) engine = InnoDB
    auto_increment = 1
    character set = utf8mb4
    collate = utf8mb4_general_ci
    row_format = Dynamic comment ='认证表';

# 验证表
drop table if exists verify;
create table if not exists verify
(
    id         int unsigned primary key auto_increment comment '主键id',
    send_id int unsigned not null comment '发送者id',
    rev_id int unsigned not null comment '接受者统称id',
    status tinyint unsigned not null default 1 comment '处理状态：1待处理，2成功，3失败',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='验证表';


# 好友表
drop table if exists friend_v2;
create table if not exists friend_v2
(
    id         int unsigned primary key auto_increment comment '主键id',
    send_id int unsigned not null comment '发送者id',
    rev_id int unsigned not null comment '接受者统称id',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='好友表v2';


# 会话表
drop table if exists conversation;
create table if not exists conversation
(
    id bigint unsigned primary key comment '会话id，雪花id',
    type tinyint not null default 1 comment '1-单聊,2-群聊',
    remark varchar(128) not null default '' comment '备注',
    pair_key varchar(32) null unique comment '单聊配对唯一键，群聊为null,解决并发竞争',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='聊天会话表';


# 会话表
drop table if exists conversation_member ;
create table if not exists conversation_member
(
    conv_id bigint unsigned not null comment '会话id，雪花id',
    user_id int unsigned not null comment '用户id',
    role tinyint not null default 1 comment '1-成员，2-管理员',
    last_read_seq bigint unsigned not null default 0 comment '最后读到的会话内序号',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree,
    primary key (conv_id, user_id),
    key idx_user_conv (user_id, conv_id)
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='会话成员表';


# 会话表
drop table if exists message ;
create table if not exists message
(
    conv_id bigint unsigned not null comment '会话id，雪花id',
    seq bigint unsigned not null comment '会话内序号,严格递增',
    msg_id bigint unsigned not null comment '全局消息id，雪花id，客户端重试不会插重复',
    send_id int unsigned not null comment '发送者id',
    content text not null comment '发送内容',
    content_type tinyint not null default 1 comment '1-文本，2-图片，3-文件',

    status tinyint not null default 1 comment '1-正常,2-撤回',
    recall_id int unsigned null comment '发送者id',
    recall_at datetime     null comment '撤回时间',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree,
    primary key (conv_id, seq),
    unique key uk_msg (msg_id),
    key idx_conv_time (conv_id, created_at, seq),
    key idx_send_time (send_id, created_at),
    key idx_conv_status (conv_id, status, seq)
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='聊天消息表';

# 会话内序号表
drop table if exists conversation_counter;
create table if not exists conversation_counter
(
    conv_id bigint unsigned primary key comment '会话id，雪花id',
    latest_seq bigint unsigned not null default 0 comment '当前会话最大seq',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='会话内序号表';




