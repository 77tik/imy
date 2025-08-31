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


# 验证表
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
    id         int unsigned primary key auto_increment comment '主键id',

    cov_id varchar(128) not null default '' comment '会话id',
    send_id int unsigned not null default '' comment '发送者userId',


    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='聊天会话表';


# 群聊表
drop table if exists group_relation;
create table if not exists group_relation
(
    id         int unsigned primary key auto_increment comment '主键id',
    group_id varchar(128) not null default '' comment '群聊id',
    user_id int unsigned not null comment '成员id',
    cov_id varchar(128) not null default '' comment '会话id',
    unread int unsigned not null default 0 comment '未读消息数',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='群聊关系表';


# 单聊表，成为好友以后要插入两条数据，分别是以对方为main_id 的会话
drop table if exists single_relation;
create table if not exists single_relation
(
    id         int unsigned primary key auto_increment comment '主键id',

    main_id int unsigned not null comment '发起成员id',
    sub_id int unsigned not null comment '接受成员id',
    cov_id varchar(128) not null default '' comment '会话id',

    created_at datetime     not null default now() comment '数据插入时间',
    updated_at datetime     not null default now() on update now() comment '数据更新时间,最后一次登陆时间',
    deleted_at tinyint(1)   not null default 0 comment '删除标记',
    index idx_deleted_at (deleted_at) using btree
) engine = InnoDB
  auto_increment = 1
  character set = utf8mb4
  collate = utf8mb4_general_ci
  row_format = Dynamic comment ='单聊关系表';

