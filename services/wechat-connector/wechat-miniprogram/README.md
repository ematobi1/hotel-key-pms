# WeChat Mini Program — Hotel Key Guest App (China)

This directory is a placeholder for the WeChat Mini Program (小程序) front end.

## Scope

- Guest-facing booking portal (China market)
- Room-service ordering
- Folio view and payment via WeChat Pay
- Push notifications via WeChat template messages

## Setup

1. Register a Mini Program at [mp.weixin.qq.com](https://mp.weixin.qq.com)
2. Obtain `AppID` and `AppSecret` — store in AWS Secrets Manager under `hotel-key/prod/wechat/`
3. Configure the service domain to point at the `wechat-connector` service endpoint
4. Use [WeChat DevTools](https://developers.weixin.qq.com/miniprogram/dev/devtools/download.html) to develop and upload

## Key APIs used

| API | Purpose |
|-----|---------|
| `wx.login` | OAuth login → code → server-side exchange for `openid` |
| `wx.requestPayment` | WeChat Pay integration |
| `subscribeMessage.send` | Push notification delivery |
| `wx.navigateTo` / `wx.redirectTo` | Page navigation |

## Directory structure (to be created)

```
wechat-miniprogram/
├── app.js
├── app.json
├── app.wxss
└── pages/
    ├── index/
    ├── booking/
    ├── folio/
    └── profile/
```
