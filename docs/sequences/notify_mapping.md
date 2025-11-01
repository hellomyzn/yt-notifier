# シーケンス — カテゴリ→出力先/ENV マッピング


```mermaid
sequenceDiagram
autonumber
participant Cfg as config/app.yaml
participant NS as NotifyService
participant Env as Environment
participant WB as Webhook


NS->>Cfg: category_to_output[category]
Cfg-->>NS: "discord" or "slack"
NS->>Cfg: category_to_env[category]
Cfg-->>NS: ENV名（例: SLACK_WEBHOOK_NEWS）
NS->>Env: getenv(ENV名)
Env-->>NS: 実際のWebhook URL
NS->>WB: POST (payload)
WB-->>NS: 2xx/4xx/5xx
```
