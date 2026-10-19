# fpack

[English](https://github.com/Matkurban/fpack/blob/main/README.md) · **中文**

### 📖 [完整文档 → matkurban.github.io/fpack](https://matkurban.github.io/fpack/?lang=zh)

各平台指南（签名、公证、安装包）、全部命令与配置键、CI 示例和问题排查。带完整注释 `fpack.yaml` 的示例应用见 [`example/`](https://github.com/Matkurban/fpack/tree/main/example)。

**一条命令，把 Flutter 项目打包成所有平台的发布文件。** 零配置可用，直接读取 `pubspec.yaml`、Gradle、Xcode 工程中的信息，绝不修改你的项目文件。

## 安装

需要 Dart 3.8+（随 Flutter 一起提供）。

```bash
dart pub global activate fpack
```

确保 `~/.pub-cache/bin`（Windows：`%LOCALAPPDATA%\Pub\Cache\bin`）在 `PATH` 中，然后检查：

```bash
fpack --version
```

第一次运行会下载与版本匹配的原生核心（约 4 MB，已校验 SHA-256）。离线和镜像方式见[安装文档](https://matkurban.github.io/fpack/installation/?lang=zh)。

## 快速开始

```bash
cd my_flutter_app
fpack init                       # （可选）生成带注释的 fpack.yaml；--yes 使用检测到的默认值
fpack doctor                     # 这台机器能打哪些包？缺什么？怎么装？
fpack build apk --dry-run        # 只显示将执行的命令，不运行
fpack build apk                  # 第一次构建 → dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk
fpack build --all                # 本机能打的全部打出来，打不了的说明原因并跳过
```

产物默认放在 `dist/<版本>+<构建号>/`，同时生成 `SHA256SUMS`。已存在的产物默认不会被覆盖：请提升版本号、使用 `--build-number N`，或加 `--force`。

## 最小的 fpack.yaml

所有键都是可选的。下面这些就足以构建并签名 Android 发布包：

```yaml
app:
  display_name: 我的应用
build:
  targets: [apk, aab]                # 不带目标运行 `fpack build` 时构建这些
android:
  signing:
    store_file: ~/keys/upload.jks
    store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}   # 不要把密码写进文件
    key_alias: upload
```

## 支持的目标

Android：`apk` · `aab` — iOS：`ipa` — macOS：`macos` · `dmg` · `pkg` — Windows：`windows` · `exe` · `msix` — Linux：`linux` · `deb` · `rpm` · `appimage` · `snap` — Web：`web`。

`fpack list` 会列出所有目标，以及这台机器能否构建。详细说明和产物命名见[目标与产物](https://matkurban.github.io/fpack/targets/?lang=zh)。

## 更多文档

其余内容都在文档网站：**[matkurban.github.io/fpack](https://matkurban.github.io/fpack/?lang=zh)**

- [快速入门](https://matkurban.github.io/fpack/getting-started/?lang=zh) · [安装](https://matkurban.github.io/fpack/installation/?lang=zh) · [命令](https://matkurban.github.io/fpack/commands/?lang=zh) · [目标与产物](https://matkurban.github.io/fpack/targets/?lang=zh)
- [配置参考](https://matkurban.github.io/fpack/configuration/?lang=zh) · [环境变量](https://matkurban.github.io/fpack/environment/?lang=zh)
- 平台指南（签名、公证、安装包）：[Android](https://matkurban.github.io/fpack/platforms/android/?lang=zh) · [iOS](https://matkurban.github.io/fpack/platforms/ios/?lang=zh) · [macOS](https://matkurban.github.io/fpack/platforms/macos/?lang=zh) · [Windows](https://matkurban.github.io/fpack/platforms/windows/?lang=zh) · [Linux](https://matkurban.github.io/fpack/platforms/linux/?lang=zh) · [Web](https://matkurban.github.io/fpack/platforms/web/?lang=zh)
- [CI 示例](https://matkurban.github.io/fpack/ci/?lang=zh) · [问题排查](https://matkurban.github.io/fpack/troubleshooting/?lang=zh) · [常见问题](https://matkurban.github.io/fpack/faq/?lang=zh)
- [AI 智能体技能](https://matkurban.github.io/fpack/skills/?lang=zh) · [架构](https://matkurban.github.io/fpack/architecture/?lang=zh) · [更新日志](https://matkurban.github.io/fpack/changelog/?lang=zh)
- 源码与问题反馈：[GitHub](https://github.com/Matkurban/fpack) · [pub.dev](https://pub.dev/packages/fpack)

## 许可证

MIT
