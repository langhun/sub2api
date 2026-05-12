# datamanagementd 已废弃

`datamanagementd` 是早期“数据管理”功能使用的独立宿主机守护进程。当前仓库已移除对应源码目录，主程序不再提供可部署的 `datamanagementd` 构建目标。

## 当前结论

- 不要再安装或启动 `sub2api-datamanagementd.service`
- 不要再挂载 `/tmp/sub2api-datamanagement.sock`
- `deploy/install-datamanagementd.sh` 仅保留废弃提示，会直接退出
- `deploy/sub2api-datamanagementd.service` 仅保留为防误用提示，不应复制到生产环境

## 旧部署的处理

如果服务器曾经安装过旧版服务，请停用并删除：

```bash
sudo systemctl disable --now sub2api-datamanagementd || true
sudo rm -f /etc/systemd/system/sub2api-datamanagementd.service
sudo systemctl daemon-reload
```

确认主程序的 Docker Compose 或 systemd 配置中不再挂载或引用：

```text
/tmp/sub2api-datamanagement.sock
```

## 恢复要求

如后续需要重新提供数据管理能力，必须先在代码中恢复明确的服务端实现、配置项、健康检查与测试，再新增部署文档。不要直接复用旧脚本或旧 systemd 单元。
