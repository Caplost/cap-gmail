# 🏗️ 系统架构图

```mermaid
graph TB
    subgraph "📧 邮件拉取与主题转发系统架构"
        subgraph "🌐 API层"
            A[REST API<br/>Gin Router]
            A1[转发目标管理 API]
            A2[邮件处理 API]
            A3[监听管理 API]
            A --> A1
            A --> A2
            A --> A3
        end
        
        subgraph "📋 业务逻辑层"
            B[ForwardingService<br/>转发服务]
            B1[EmailParser<br/>邮件解析]
            B2[EmailMonitor<br/>自动监听]
            B --> B1
            B --> B2
        end
        
        subgraph "📨 邮件服务层"
            C1[IMAP Service<br/>邮件接收]
            C2[SMTP Service<br/>邮件发送]
            C3[Gmail API Service<br/>Gmail集成]
        end
        
        subgraph "🗄️ 数据层"
            D1[(MySQL数据库)]
            D2[ForwardingTarget<br/>转发目标表]
            D3[EmailLog<br/>邮件日志表]
            D1 --> D2
            D1 --> D3
        end
        
        subgraph "🔧 工具层"
            E1[EmailDecoder<br/>邮件解码]
            E2[Security Utils<br/>安全工具]
            E3[Logger<br/>日志工具]
        end
    end
    
    subgraph "📥 外部系统"
        F1[Gmail IMAP<br/>邮件接收]
        F2[Gmail SMTP<br/>邮件发送]
        F3[Gmail API<br/>OAuth认证]
    end
    
    A1 --> B
    A2 --> B
    A3 --> B2
    
    B --> C1
    B --> C2
    B --> C3
    B --> D1
    
    C1 --> F1
    C2 --> F2
    C3 --> F3
    
    C1 --> E1
    C2 --> E1
    C3 --> E1
    
    B --> E2
    B --> E3
```

## 架构说明

### 🌐 API层
- **REST API**: 基于Gin框架的RESTful接口
- **转发目标管理**: CRUD操作转发规则
- **邮件处理**: 手动触发邮件处理
- **监听管理**: 启动/停止自动监听

### 📋 业务逻辑层  
- **ForwardingService**: 核心转发逻辑，包含去重机制
- **EmailParser**: 邮件主题解析和关键词匹配
- **EmailMonitor**: 自动监听新邮件

### 📨 邮件服务层
- **IMAP Service**: 连接Gmail IMAP接收邮件
- **SMTP Service**: 通过SMTP发送邮件
- **Gmail API Service**: 使用Gmail API收发邮件

### 🗄️ 数据层
- **MySQL**: 生产级数据库
- **ForwardingTarget**: 转发目标配置
- **EmailLog**: 邮件处理日志

### 🔧 工具层
- **EmailDecoder**: 处理中文编码和HTML清理
- **Security Utils**: 数据脱敏和安全功能
- **Logger**: 结构化日志 