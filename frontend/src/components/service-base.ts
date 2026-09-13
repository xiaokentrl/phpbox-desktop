// 服务线模块元数据契约：每个服务模块（php/mysql/pgsql/redis/nginx）一份 meta，
// ServiceBase 按它渲染版本卡网格；差异点（如 PHP 的扩展按钮）经 slot 注入。
export interface ServiceMeta {
  svc: string                  // 服务键（nav/i18n/install CLI 参数同构）
  icon: string                 // 空态图标
  suggested: string[]          // 安装弹窗版本快选
  single?: boolean             // 单实例（nginx：install 无版本参数）
  portMap?: Record<string, string> // 版本 → 端口展示
  hasPassword?: boolean        // 卡片显示密码行（mysql/pgsql）
}
