# 用户相关接口

## 1 根据邮箱获取验证码

### 1.1 接口描述
获取用户邮箱验证码

### 1.2 请求方法
GET user/send-code

### 1.3 输入参数
只需传入用户邮箱

### 1.4 输出参数
code string 状态码
message string 状态描述

### 1.5 请求示例
#### 输入示例
curl http://127.0.0.1:9000/user/send-code?email=1512619344@qq.com
#### 输出示例
{
    "code":"邮箱验证码",
    "message":"success"
}

## 2 用户注册接口

### 2.1 接口描述
用户注册

### 2.2 请求方法
POST user/register

### 2.3 输入参数
nickname string 用户名
email string 用户邮箱
password string 用户密码
repeat_password string 确认用户密码
verify_code string 邮箱验证码

### 2.4 输出参数
code string 状态码
message string 状态描述

### 2.5 请求示例
#### 输入示例
POST http://localhost:9000/user/register HTTP/1.1
Content-Type: application/json

{
    "nickname": "wtf",
    "email": "2117907739@qq.com",
    "password": "123456aB",
    "repeat_password": "123456aB",
    "verify_code": "8e7e"
}
#### 输出示例
{
  "code": 200,
  "message": "register successful"
}

## 3 用户登录接口

### 3.1 账号密码登录

#### 3.1.1 接口描述
用户登录

#### 3.1.2 请求方法
POST user/login

#### 3.1.3 输入参数
nickname string 用户名
password string 用户密码

#### 3.1.4 输出参数
code string 状态码
message string 状态描述

#### 3.1.5 请求示例
##### 输入示例
POST http://localhost:9000/user/login HTTP/1.1
Content-Type: application/json

{
    "nickname": "wtf",
    "password": "123456aB"
}
##### 输出示例
{
  "code": 200,
  "message": "login successful",
  "token": "JWTTOKEN"
}

### 3.2 邮箱密码登录

#### 3.2.1 接口描述
用户登录

#### 3.2.2 请求方法
POST user/login

#### 3.2.3 输入参数
email string 用户邮箱
password string 用户密码

#### 3.2.4 输出参数
code string 状态码
message string 状态描述

#### 3.2.5 请求示例
##### 输入示例
POST http://localhost:9000/user/login HTTP/1.1
Content-Type: application/json

{
    "email": "2117907739@qq.com",
    "password": "123456aB"
}
##### 输出示例
{
  "code": 200,
  "message": "login successful",
  "token": "JWTTOKEN"
}

### 3.3 邮箱验证码登录

#### 3.2.1 接口描述
用户登录

#### 3.2.2 请求方法
POST user/login

#### 3.2.3 输入参数
email string 用户邮箱
password string 用户密码

#### 3.2.4 输出参数
code string 状态码
message string 状态描述

#### 3.2.5 请求示例
##### 输入示例
POST http://localhost:9000/user/login HTTP/1.1
Content-Type: application/json

{
    "email": "2117907739@qq.com",
    "verify_code": "123456aB"
}
##### 输出示例
{
  "code": 200,
  "message": "login successful",
  "token": "JWTTOKEN"
}

## 4 用户登出接口

### 4.1 接口描述
用户登出

### 4.2 请求方法
GET user/logout

### 4.3 输入参数
无

### 4.4 输出参数
code string 状态码
message string 状态描述

### 4.5 请求示例
#### 输入示例
GET http://localhost:9000/user/logout HTTP/1.1
Authorization: Bearer JWTTOKEN
#### 输出示例
{
  "code": 200,
  "message": "logout successful"
}

## 5 获取用户信息

### 5.1 接口描述
获取用户信息

### 5.2 请求方法
GET user/get-userinfo

### 5.3 输入参数
无

### 5.4 输出参数
uuid string 用户唯一标识
nickname string 用户名
email string 用户邮箱
avatar string 用户头像

### 5.5 请求示例
#### 输入示例
GET http://localhost:9000/user/get-userinfo HTTP/1.1
Authorization: Bearer JWTTOKEN
#### 输出示例
{
  "uuid": "0b1da72b-abfa-46ff-9a0e-b3c1442b76bc",
  "nickname": "wtf",
  "email": "2117907739@qq.com",
  "avatar": "/static/avatar/default.png"
}

## 6 更新用户信息

### 6.1 接口描述
更新用户信息

### 6.2 请求方法
POST user/update-userinfo

### 6.3 输入参数
nickname string 用户名
avatar string 用户头像

### 6.4 输出参数
code string 状态码
message string 状态描述

### 6.5 请求示例
#### 输入示例
PUT http://localhost:9000/user/update-userinfo
Authorization: Bearer JWTTOKEN
Content-Type: multipart/form-data; boundary=WebKitFormBoundary7MA4YWxkTrZu0gW

--WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="nickname"

wtf2
--WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="avatar"; filename="changeavatar.JPG"
Content-Type: image/jpeg

< /root/projects/github/AI-Nexus/changeavatar.jpg
#### 输出示例
{
  "code": 200,
  "message": "update userinfo successful"
}