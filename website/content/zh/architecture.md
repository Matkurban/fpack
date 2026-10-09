---
title: "架构与开发"
description: "fpack 的结构，以及如何开发和发布。"
lang: zh-CN
---

```
fpack/
  bin/fpack.dart            Dart 入口（pub executable）
  lib/src/                  启动器：平台识别、核心解析（缓存 → 预编译 → 下载 → 本机编译）、SHA-256、信号转发
  go/                       原生核心（Go，依赖已 vendor）
    cmd/fpack-core/         main
    internal/cli            参数解析、命令、帮助
    internal/build          构建编排：分类 → 预检 → 共享 flutter 步骤 → 打包 → 校验和 → 汇总
    internal/targets        每个目标的预检、flutter 参数、产物定位、打包命令（dry-run 与真实执行同一份代码）
    internal/config         键注册表（驱动环境变量、校验、JSON Schema、init 模板与文档表格）、fpack.yaml 解析
    internal/project        pubspec / Gradle / Xcode 工程信息读取（只读）
    internal/flutter        Flutter SDK 查找（FVM / PATH / …）
    internal/runner         子进程（进程组、Ctrl-C、日志、密钥脱敏）
    internal/hints          常见错误 → 修复建议
    internal/ui, i18n, pack, host, doctor, version
  schema/                   fpack.schema.json（由键注册表生成）
  prebuilt/                 预编译二进制 + manifest.json（发布产物，不进 git）
  scripts/                  build_binaries.sh、check_versions.sh、package_dist.sh
  .github/workflows/        ci.yml、e2e.yml、release.yml
```

## 开发与发布

```bash
cd go && go vet ./... && go test ./...      # Go 核心
FPACK_UPDATE=1 go test ./internal/config    # 修改键注册表后重新生成 schema 与 doc/configuration.md 表格
dart pub get && dart analyze && dart test   # Dart 启动器
scripts/build_binaries.sh                   # 交叉编译 6 个平台 → prebuilt/
scripts/package_dist.sh ../fpack-dist.tar.gz  # 自包含离线安装包
FPACK_CORE=$PWD/prebuilt/darwin-arm64/fpack-core fpack doctor   # 用指定核心调试
```

发布：同时修改 `pubspec.yaml`、`lib/src/version.dart`、`go/internal/version/version.go` 中的版本号（`scripts/check_versions.sh` 会检查），更新 `CHANGELOG.md`，推送 `vX.Y.Z` 标签。`release.yml` 会交叉编译、创建 GitHub Release（二进制 + `checksums.txt` + 离线包），并可选发布到 pub.dev。
