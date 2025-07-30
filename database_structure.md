# 📊 Gmail邮件拉取与主题转发系统 - 数据库结构

## 📋 数据库表概览

| 表名 | 中文名 | 用途 | 记录数预估 |
|------|--------|------|------------|
| `forwarding_targets` | 转发目标表 | 存储邮件转发的目标配置 | 10-100 |
| `email_logs` | 邮件处理日志表 | 记录每封邮件的处理结果 | 1000+ |

---

## 🎯 1. forwarding_targets (转发目标表)

### 表结构
```sql
CREATE TABLE `forwarding_targets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '转发对象名字',
  `email` varchar(255) NOT NULL COMMENT '转发邮箱地址',
  `keywords` text COMMENT '关联的关键字，逗号分隔',
  `is_active` tinyint(1) DEFAULT '1' COMMENT '是否启用',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 字段说明
| 字段名 | 类型 | 长度 | 允许空 | 默认值 | 说明 |
|--------|------|------|--------|--------|------|
| `id` | bigint unsigned | - | NO | AUTO_INCREMENT | 主键ID |
| `name` | varchar | 255 | NO | - | 转发对象名字（如：技术支持） |
| `email` | varchar | 255 | NO | - | 转发邮箱地址 |
| `keywords` | text | - | YES | NULL | 关联关键字（逗号分隔） |
| `is_active` | tinyint(1) | - | YES | 1 | 是否启用 |
| `created_at` | datetime(3) | - | YES | NULL | 创建时间 |
| `updated_at` | datetime(3) | - | YES | NULL | 更新时间 |

### 示例数据
```json
{
  "id": 1,
  "name": "技术支持",
  "email": "leonscottcap@gmail.com",
  "keywords": "技术支持,技术故障,系统问题,bug,错误",
  "is_active": true,
  "created_at": "2025-07-30T16:00:00Z",
  "updated_at": "2025-07-30T16:00:00Z"
}
```

---

## 📧 2. email_logs (邮件处理日志表)

### 表结构
```sql
CREATE TABLE `email_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `message_id` varchar(255) NOT NULL COMMENT '邮件唯一标识',
  `subject` text COMMENT '邮件主题',
  `from_email` varchar(255) COMMENT '发件人邮箱',
  `to_email` varchar(255) COMMENT '收件人邮箱',
  `forwarded_to` varchar(255) COMMENT '转发到的邮箱',
  `keyword` varchar(255) COMMENT '匹配的关键字',
  `target_name` varchar(255) COMMENT '转发对象名字',
  `status` varchar(20) DEFAULT 'pending' COMMENT '处理状态',
  `error_msg` text COMMENT '错误信息',
  `processed_at` datetime(3) COMMENT '处理时间',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_email_logs_message_target` (`message_id`,`forwarded_to`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 字段说明
| 字段名 | 类型 | 长度 | 允许空 | 默认值 | 说明 |
|--------|------|------|--------|--------|------|
| `id` | bigint unsigned | - | NO | AUTO_INCREMENT | 主键ID |
| `message_id` | varchar | 255 | NO | - | 邮件唯一标识 |
| `subject` | text | - | YES | NULL | 邮件主题 |
| `from_email` | varchar | 255 | YES | NULL | 发件人邮箱 |
| `to_email` | varchar | 255 | YES | NULL | 收件人邮箱 |
| `forwarded_to` | varchar | 255 | YES | NULL | 转发到的邮箱 |
| `keyword` | varchar | 255 | YES | NULL | 匹配的关键字 |
| `target_name` | varchar | 255 | YES | NULL | 转发对象名字 |
| `status` | varchar | 20 | YES | 'pending' | 处理状态 |
| `error_msg` | text | - | YES | NULL | 错误信息 |
| `processed_at` | datetime(3) | - | YES | NULL | 处理时间 |
| `created_at` | datetime(3) | - | YES | NULL | 创建时间 |
| `updated_at` | datetime(3) | - | YES | NULL | 更新时间 |

### 状态枚举 (EmailStatus)
| 状态值 | 中文描述 | 说明 |
|--------|----------|------|
| `pending` | 待处理 | 邮件已接收，等待处理 |
| `forwarded` | 已转发 | 邮件成功转发 |
| `failed` | 失败 | 转发失败 |
| `skipped` | 跳过 | 不匹配关键字或已处理 |

### 示例数据
```json
{
  "id": 1,
  "message_id": "<uuid@example.com>",
  "subject": "技术故障修复会议 - 技术支持",
  "from_email": "capyinneng@gmail.com",
  "to_email": "wangyinneng@gmail.com",
  "forwarded_to": "leonscottcap@gmail.com",
  "keyword": "技术支持",
  "target_name": "技术支持",
  "status": "forwarded",
  "error_msg": null,
  "processed_at": "2025-07-30T16:05:00Z",
  "created_at": "2025-07-30T16:05:00Z",
  "updated_at": "2025-07-30T16:05:00Z"
}
```

---

## 🔑 重要约束和索引

### 1. 复合唯一约束
- **表**: `email_logs`
- **约束**: `uni_email_logs_message_target (message_id, forwarded_to)`
- **作用**: 防止同一邮件重复转发给同一目标，实现去重功能

### 2. 业务规则
1. **转发目标**: 同一邮箱地址可以对应多个转发目标（一对多转发）
2. **去重机制**: 同一邮件（MessageID）可以转发给多个不同目标，但不能重复转发给同一目标
3. **关键词匹配**: 支持模糊匹配和严格匹配两种模式
4. **状态追踪**: 完整记录邮件处理的每个环节

---

## 📈 数据增长预估

| 表名 | 初始数据 | 日增长 | 月增长 | 年增长 |
|------|----------|--------|--------|--------|
| `forwarding_targets` | 10-20条 | 1-2条 | 30条 | 100条 |
| `email_logs` | 0条 | 50-200条 | 1500-6000条 | 18K-72K条 |

---

## 🔧 维护建议

1. **定期清理**: 建议每年清理一次超过1年的 `email_logs` 数据
2. **索引优化**: 根据查询频率为常用字段添加索引
3. **分区策略**: 如果数据量很大，可考虑按时间分区
4. **备份策略**: 重要的 `forwarding_targets` 表需要定期备份 