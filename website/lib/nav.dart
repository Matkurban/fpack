import 'package:jaspr_content/components/sidebar.dart';

/// Sidebar navigation for one language. [p] is the URL prefix ('' or '/zh').
List<SidebarGroup> sidebarGroups(String lang) {
  final zh = lang == 'zh';
  final p = zh ? '/zh' : '';
  String t(String en, String cn) => zh ? cn : en;
  return [
    SidebarGroup(
      links: [
        SidebarLink(text: t('Overview', '概览'), href: '$p/'),
        SidebarLink(text: t('Getting started', '快速开始'), href: '$p/getting-started'),
        SidebarLink(text: t('Installation', '安装'), href: '$p/installation'),
      ],
    ),
    SidebarGroup(
      title: t('Reference', '参考'),
      links: [
        SidebarLink(text: t('Commands', '命令'), href: '$p/commands'),
        SidebarLink(text: t('Targets and artifacts', '目标与产物'), href: '$p/targets'),
        SidebarLink(text: t('Configuration', '配置'), href: '$p/configuration'),
        SidebarLink(text: t('Environment variables', '环境变量'), href: '$p/environment'),
      ],
    ),
    SidebarGroup(
      title: t('Platform guides', '平台指南'),
      links: [
        SidebarLink(text: 'Android', href: '$p/platforms/android'),
        SidebarLink(text: 'iOS', href: '$p/platforms/ios'),
        SidebarLink(text: t('macOS (signing, notarization)', 'macOS（签名、公证）'), href: '$p/platforms/macos'),
        SidebarLink(text: 'Windows', href: '$p/platforms/windows'),
        SidebarLink(text: 'Linux', href: '$p/platforms/linux'),
        SidebarLink(text: 'Web', href: '$p/platforms/web'),
      ],
    ),
    SidebarGroup(
      title: t('More', '更多'),
      links: [
        SidebarLink(text: t('CI recipes', 'CI 示例'), href: '$p/ci'),
        SidebarLink(text: t('AI agent skill', 'AI 智能体技能'), href: '$p/skills'),
        SidebarLink(text: t('Troubleshooting', '故障排查'), href: '$p/troubleshooting'),
        SidebarLink(text: t('FAQ', '常见问题'), href: '$p/faq'),
        SidebarLink(text: t('Architecture', '架构'), href: '$p/architecture'),
        SidebarLink(text: t('Changelog', '更新日志'), href: '$p/changelog'),
      ],
    ),
  ];
}
