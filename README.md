# 微助教交互式签到

用 Go 实现的针对微助教的自动化签到服务

## 特性

- 支持普通签到、GPS 签到、二维码签到
- 邮件提醒
- 自定义轮询间隔和拟真延迟时间
- 包含前端页面

## TODO

- [ ] GPS 签到添加真正的经纬度
- [ ] OpenID 失效提醒
- [ ] OpenID 自动解析

## 用法

### 配置文件

```yml
# 配置一个Redis地址
redis:
  address: "localhost:6379"
  password: "RedisPassword"
  db: 0

# 服务器参数
app:
  # 检测周期间隔
  interval: 8
  # 二维码签到检测到后延迟秒数
  normal_delay: 20
  # 服务器监听地址
  url: http://localhost:8080

# 发送邮件反馈
mail:
  # 是否启用
  enabled: true
  host: smtp.example.com
  port: 465
  username: admin@example.com
  password: MailPassword
  from: admin@example.com
```

### 首页

http://example.com:8080/home
包含一个要求填入 OpenID 和邮箱的表单。

### 登记 OpenID

在微助教学生页面**加载出来之前**点击右上角三个点，选择到浏览器打开，这个链接带有 openid 字段。复制 32 位 OpenID 的值到表单中，填入自己的邮箱，点击提交即可。**所有 OpenID 都只有 2 小时的有效期。提交完成后不能再次打开任何微助教页面，这会使 OpenID 立刻失效。** 所以你可以通过新打开微助教的页面来使原先的**提前失效**。

### 服务器状态

可以获得服务器当前正在监控的所有 OpenID、轮询间隔和拟真延迟时间

### 二维码签到

普通签到和 GPS 签到在成功添加监控之后就不用做任何事了，签到成功会有邮件提醒。
二维码签到则会立刻发送一份带有链接的邮件，链接包含和微助教同步实时更新的二维码，用微信扫一扫即可。**扫码后原来的 OpenID 会立刻失效，如果需要多次签到必须要重新完成上述流程。**
