基于 Go 实现的 SignalRoute 项目，一款城轨信号安全联锁服务，协调进路、道岔、轨道区段与信号机的失效安全状态。

SignalRoute 使用本地事件日志和原子快照保存控制状态，提供进路、道岔、信号、轨道区段、道口、轴计数器和分区供电接口，并附带四个运营控制页面。

使用固定 Go 工具链离线构建：

```text
go build -mod=vendor ./...
```

启动服务：

```text
signalroute -addr 127.0.0.1:21211 -data ./signalroute-data
```

健康入口为 `/healthz`，控制页面为 `/routes`、`/points`、`/signals` 和 `/incidents`。
