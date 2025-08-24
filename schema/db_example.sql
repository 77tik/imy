-- 用户表
CREATE TABLE `user_models` (
                               `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                               `created_at` datetime DEFAULT NULL,
                               `updated_at` datetime DEFAULT NULL,

                               `uuid` varchar(64) NOT NULL,
                               `nick_name` varchar(32) DEFAULT NULL,
                               `password` varchar(128) NOT NULL,
                               `email` varchar(128) DEFAULT NULL,
                               `file_name` varchar(256) DEFAULT 'eb3dad2d-4b7f-44c2-9af5-50ad9f76ff81.png',
                               `abstract` varchar(128) DEFAULT NULL,
                               `phone` varchar(11) DEFAULT NULL,
                               `status` tinyint DEFAULT '1',
                               `gender` tinyint DEFAULT '3',
                               `last_login_ip` varchar(39) DEFAULT NULL,
                               `source` int DEFAULT NULL,

                               PRIMARY KEY (`id`),
                               UNIQUE KEY `uuid` (`uuid`),
                               KEY `idx_user_models_nick_name` (`nick_name`),
                               KEY `idx_user_models_email` (`email`),
                               KEY `idx_user_models_phone` (`phone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 好友关系表
CREATE TABLE `friend_models` (
                                 `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                 `created_at` datetime DEFAULT NULL,
                                 `updated_at` datetime DEFAULT NULL,

                                 `send_user_id` varchar(64) DEFAULT NULL,
                                 `rev_user_id` varchar(64) DEFAULT NULL,
                                 `send_user_notice` varchar(128) DEFAULT NULL,
                                 `rev_user_notice` varchar(128) DEFAULT NULL,
                                 `source` varchar(32) DEFAULT NULL,
                                 `is_deleted` tinyint(1) NOT NULL DEFAULT '0',

                                 PRIMARY KEY (`id`),
                                 KEY `idx_friend_models_send_user_id` (`send_user_id`),
                                 KEY `idx_friend_models_rev_user_id` (`rev_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 好友验证表
CREATE TABLE `friend_verify_models` (
                                        `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                        `created_at` datetime DEFAULT NULL,
                                        `updated_at` datetime DEFAULT NULL,

                                        `send_user_id` varchar(64) DEFAULT NULL,
                                        `rev_user_id` varchar(64) DEFAULT NULL,
                                        `send_status` tinyint DEFAULT NULL,
                                        `rev_status` tinyint DEFAULT NULL,
                                        `message` varchar(128) DEFAULT NULL,
                                        `source` varchar(32) DEFAULT NULL,

                                        PRIMARY KEY (`id`),
                                        KEY `idx_friend_verify_models_send_user_id` (`send_user_id`),
                                        KEY `idx_friend_verify_models_rev_user_id` (`rev_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 群组表
CREATE TABLE `group_models` (
                                `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                `created_at` datetime DEFAULT NULL,
                                `updated_at` datetime DEFAULT NULL,

                                `uuid` varchar(64) NOT NULL,
                                `type` tinyint DEFAULT '1',
                                `title` varchar(32) DEFAULT NULL,
                                `abstract` varchar(128) DEFAULT NULL,
                                `file_name` varchar(256) DEFAULT '71e4be6c-b477-4fce-8348-9cc53349ef28.png',
                                `creator_id` varchar(64) DEFAULT NULL,
                                `notice` text,
                                `tags` varchar(256) DEFAULT NULL,
                                `max_members` int DEFAULT '500',
                                `current_members` int DEFAULT '0',
                                `status` tinyint DEFAULT '1',
                                `mute_all` tinyint(1) DEFAULT '0',
                                `dissolve_time` datetime DEFAULT NULL,
                                `category` varchar(32) DEFAULT NULL,
                                `join_auth` tinyint DEFAULT '1',
                                `member_invite` tinyint(1) DEFAULT '1',
                                `member_manage` tinyint(1) DEFAULT '0',
                                `message_archive` tinyint(1) DEFAULT '1',
                                `allow_view_history` tinyint(1) DEFAULT '1',

                                PRIMARY KEY (`id`),
                                UNIQUE KEY `uuid` (`uuid`),
                                KEY `idx_group_models_title` (`title`),
                                KEY `idx_group_models_creator_id` (`creator_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 群组成员表
CREATE TABLE `group_member_models` (
                                       `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                       `created_at` datetime DEFAULT NULL,
                                       `updated_at` datetime DEFAULT NULL,

                                       `group_id` varchar(64) DEFAULT NULL,
                                       `user_id` varchar(64) DEFAULT NULL,
                                       `member_nickname` varchar(32) DEFAULT NULL,
                                       `role` tinyint DEFAULT NULL,
                                       `prohibition_time` int DEFAULT NULL,
                                       `inviter_id` varchar(64) DEFAULT NULL,
                                       `status` tinyint DEFAULT '1',
                                       `notify_level` tinyint DEFAULT '1',
                                       `display_name` varchar(32) DEFAULT NULL,

                                       PRIMARY KEY (`id`),
                                       KEY `idx_group_member_models_group_id` (`group_id`),
                                       KEY `idx_group_member_models_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 聊天消息表
CREATE TABLE `chat_models` (
                               `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                               `created_at` datetime DEFAULT NULL,
                               `updated_at` datetime DEFAULT NULL,

                               `message_id` varchar(64) DEFAULT NULL,
                               `conversation_id` varchar(64) DEFAULT NULL,
                               `send_user_id` varchar(64) DEFAULT NULL,
                               `msg_type` tinyint DEFAULT NULL,
                               `msg_preview` varchar(64) DEFAULT NULL,
                               `msg` longtext,
                               `is_deleted` tinyint(1) NOT NULL DEFAULT '0',

                               PRIMARY KEY (`id`),
                               KEY `idx_chat_models_send_user_id` (`send_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户会话表
CREATE TABLE `chat_user_conversation_models` (
                                                 `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                                 `created_at` datetime DEFAULT NULL,
                                                 `updated_at` datetime DEFAULT NULL,

                                                 `user_id` varchar(64) NOT NULL,
                                                 `conversation_id` varchar(64) NOT NULL,
                                                 `last_message` text,
                                                 `is_deleted` tinyint(1) NOT NULL DEFAULT '0',
                                                 `is_pinned` tinyint(1) NOT NULL DEFAULT '0',

                                                 PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 文件表
CREATE TABLE `file_models` (
                               `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                               `created_at` datetime DEFAULT NULL,
                               `updated_at` datetime DEFAULT NULL,

                               `file_name` varchar(64) DEFAULT NULL,
                               `original_name` varchar(128) DEFAULT NULL,
                               `size` bigint DEFAULT NULL,
                               `path` varchar(256) DEFAULT NULL,
                               `md5` varchar(32) DEFAULT NULL,
                               `type` varchar(32) DEFAULT NULL,
                               `source` varchar(32) DEFAULT 'qiniu',
                               `file_info` longtext,

                               PRIMARY KEY (`id`),
                               UNIQUE KEY `file_name` (`file_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 动态表
CREATE TABLE `moment_models` (
                                 `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                 `created_at` datetime DEFAULT NULL,
                                 `updated_at` datetime DEFAULT NULL,

                                 `user_id` varchar(64) NOT NULL,
                                 `content` text,
                                 `files` longtext,
                                 `is_deleted` tinyint(1) NOT NULL DEFAULT '0',

                                 PRIMARY KEY (`id`),
                                 KEY `idx_moment_models_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 动态评论表
CREATE TABLE `moment_comment_models` (
                                         `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                         `created_at` datetime DEFAULT NULL,
                                         `updated_at` datetime DEFAULT NULL,

                                         `user_id` varchar(64) NOT NULL,
                                         `moment_id` bigint unsigned NOT NULL,
                                         `content` text NOT NULL,
                                         `is_deleted` tinyint(1) NOT NULL DEFAULT '0',

                                         PRIMARY KEY (`id`),
                                         KEY `idx_moment_comment_models_user_id` (`user_id`),
                                         KEY `idx_moment_comment_models_moment_id` (`moment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 动态点赞表
CREATE TABLE `moment_like_models` (
                                      `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                      `created_at` datetime DEFAULT NULL,
                                      `updated_at` datetime DEFAULT NULL,

                                      `user_id` varchar(64) NOT NULL,
                                      `moment_id` bigint unsigned NOT NULL,
                                      `is_deleted` tinyint(1) NOT NULL DEFAULT '0',

                                      PRIMARY KEY (`id`),
                                      KEY `idx_moment_like_models_user_id` (`user_id`),
                                      KEY `idx_moment_like_models_moment_id` (`moment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 动态收藏表
CREATE TABLE `moment_favorite_models` (
                                          `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                          `created_at` datetime DEFAULT NULL,
                                          `updated_at` datetime DEFAULT NULL,

                                          `user_id` varchar(64) NOT NULL,
                                          `moment_id` bigint unsigned NOT NULL,

                                          PRIMARY KEY (`id`),
                                          KEY `idx_moment_favorite_models_user_id` (`user_id`),
                                          KEY `idx_moment_favorite_models_moment_id` (`moment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 动态举报表
CREATE TABLE `moment_report_models` (
                                        `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                        `created_at` datetime DEFAULT NULL,
                                        `updated_at` datetime DEFAULT NULL,

                                        `user_id` varchar(64) NOT NULL,
                                        `moment_id` bigint unsigned NOT NULL,
                                        `reason` text NOT NULL,
                                        `images` longtext,
                                        `status` int NOT NULL DEFAULT '0',

                                        PRIMARY KEY (`id`),
                                        KEY `idx_moment_report_models_user_id` (`user_id`),
                                        KEY `idx_moment_report_models_moment_id` (`moment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 反馈表
CREATE TABLE `feedback_models` (
                                   `id` bigint unsigned NOT NULL AUTO_INCREMENT,
                                   `created_at` datetime DEFAULT NULL,
                                   `updated_at` datetime DEFAULT NULL,

                                   `user_id` varchar(64) DEFAULT NULL,
                                   `content` text NOT NULL,
                                   `type` tinyint NOT NULL,
                                   `status` tinyint NOT NULL DEFAULT '1',
                                   `file_names` json DEFAULT NULL,
                                   `handler_id` bigint DEFAULT NULL,
                                   `handle_time` datetime DEFAULT NULL,
                                   `handle_result` text,
    
                                   PRIMARY KEY (`id`),
                                   KEY `idx_feedback_models_user_id` (`user_id`),
                                   KEY `idx_feedback_models_handler_id` (`handler_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;