---
title: "Windows"
description: "便携 zip、Inno Setup 安装包、MSIX 与代码签名。"
---

设置 `windows.sign.certificate`（.pfx）+ `password`，或 `thumbprint`（证书存储）后：`flutter build windows` 之后立即用 signtool 签名应用 .exe（zip、安装包、MSIX 中都是已签名的 exe），Inno Setup 安装程序和卸载程序通过 `SignTool=` 签名，MSIX 使用同一证书；时间戳服务器默认 `http://timestamp.digicert.com`，signtool 会自动在 Windows SDK 中查找。

Inno Setup 安装包可配置发布者/网址、版权、安装目录、开始菜单、桌面快捷方式、许可/信息页、图标与向导图片、最低 Windows 版本、`privileges`（`user` 免管理员 / `admin` / `ask`）以及多语言（`languages: [zh-CN, en]`，第一个为默认；旧版 Inno Setup 缺少的中文语言文件由 fpack 自带）。MSIX 的显示名、发布者、能力、文件关联、协议、版本等可直接在 `windows.msix` 中设置，优先于 pubspec 的 `msix_config`。

## 目标

| 目标 | 产物 | 需要 |
| --- | --- | --- |
| `windows` | `-windows-x64-portable.zip` | 装有「使用 C++ 的桌面开发」工作负载的 Visual Studio |
| `exe` | `-windows-x64-setup.exe` | [Inno Setup 6](https://jrsoftware.org/isinfo.php)（`ISCC.exe`；默认安装位置会自动查找，也可用 `windows.inno_setup.iscc` 指定） |
| `msix` | `-windows-x64.msix` | `dart pub add --dev msix` |

三者共用一次 `flutter build windows`。

## 安装包示例

```yaml
windows:
  inno_setup:
    languages: [zh-CN, en]      # 第一个为默认语言
    privileges: user            # user | admin | ask
    desktop_icon: checked       # checked | unchecked | none
    license_file: LICENSE.txt
  msix:
    publisher: "CN=Your Company"
    capabilities: [internetClient]
  sign:
    certificate: C:/certs/codesign.pfx
    password: ${WINDOWS_CERT_PASSWORD}
```

`windows.inno_setup.app_id` 由 `fpack init` 生成一次——请保持不变，升级安装时才能替换已安装的版本。要使用自己的 `.iss` 脚本，请设置 `windows.inno_setup.script`。
