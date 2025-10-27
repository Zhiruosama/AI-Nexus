# 1. 根据ID获取用户信息

## 1.1 接口描述
获取用户信息

## 1.2 请求方法
GET demo/get-message

## 1.3 输入参数
只需传入用户id

## 1.4 输出参数
用户ID+用户Message

## 1.5 请求示例
### 输入示例
curl http://127.0.0.1:9000/demo/get-message?id=1
### 输出示例
Begin demo{"id":2,"message":"helloworld2"}