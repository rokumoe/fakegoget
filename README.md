# fakegoget

给没梯子的孤儿们用的镜像 go get.

曲线救国 go get 某些 golang.org, googlesource.com 之类的域名.

## usage

1. 直接拉 git, 不要 go get

```
git clone github.com/rokumoe/fakegoget
```

2. 自签证书后启动 https 服务器

```
cd fakegoget
sh sslcmds
sudo ./fakegoget
```

3. 修改 hosts 把配置里域名都指向 fakegoget 监听 IP

4. 信任自签的 CA, 或者使用 `go get -insecure`
