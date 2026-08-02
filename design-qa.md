# 充值、订单与邀请返利设计验收

验收日期：2026-07-22

final result: passed

## 对照范围

- 参考页面：`https://api.gjx88.com/purchase`、`/orders`、`/affiliate`
- 用户截图：`C:\Users\ADMINI~1\AppData\Local\Temp\codex-clipboard-6e4ab6ca-3f92-4534-8ebc-3ca5af3d81a2.png`
- 实现页面：`http://127.0.0.1:3010/purchase`、`/orders`、`/affiliate`
- 桌面视口：1440 x 900
- 移动视口：390 x 844

## 视觉结论

- [x] 侧栏按用户工作流展示“充值/订阅”“我的订单”“邀请返利”，并保持悦享 API 现有浅色、品牌蓝和紧凑后台风格。
- [x] 充值页沿用现有金额输入、快捷金额、支付方式和结果摘要组件，支付关闭时用明确的筹备提示替代可提交状态。
- [x] 订单页保留筛选、刷新、返回充值和表格空状态，信息密度及操作层级与现有后台一致。
- [x] 邀请页展示 15% 比例、邀请数据、邀请码、邀请链接和返利明细，并明确说明只可转入站内余额。
- [x] 桌面和 390px 手机端均无重叠、截断或横向溢出，手机端邀请码和链接复制按钮完整可见。

## 交互结论

- [x] 支付关闭时充值与订阅表单可查看，支付宝和微信以禁用占位状态展示，创建订单按钮保持禁用且不发送请求。
- [x] 订单筛选、刷新和返回充值入口可操作。
- [x] 邀请返利转余额流程可完成，成功后可用返利清零、按钮禁用并同步增加账户余额。
- [x] 页面导航、金额选择、输入、复制及响应式切换均完成浏览器验证。

## 差异说明

- 参考站使用深色视觉，当前实现遵循悦享 API 既有浅色系统风格。
- 参考站已接入真实支付；当前支付仅展示界面，等待工商和支付服务商接入后再启用订单提交。
- 邀请返利按确认规则固定为默认 15%，只允许转站内余额，不展示现金提现入口。

## 问题分级

- P0：无。
- P1：无。
- P2：无。

---

# 模型广场价格对照版设计验收

验收日期：2026-08-02

final result: passed

## 对照范围

- 用户参考图：`C:\Users\ADMINI~1\AppData\Local\Temp\codex-clipboard-59e2573f-b5d6-402f-a426-69a625c121e0.png`
- 本地实现：`http://127.0.0.1:4173/model-plaza`
- 桌面实现图：`C:\Users\Administrator\.codex\visualizations\2026\08\02\model-plaza-redesign\desktop.png`
- 手机上半屏：`C:\Users\Administrator\.codex\visualizations\2026\08\02\model-plaza-redesign\mobile.png`
- 手机下半屏：`C:\Users\Administrator\.codex\visualizations\2026\08\02\model-plaza-redesign\mobile-lower.png`
- 并排对照：`C:\Users\Administrator\.codex\visualizations\2026\08\02\model-plaza-redesign\comparison.png`
- 桌面视口：用户 Chrome 默认桌面视口；手机视口：390 x 844。

## 对照结论

- [x] 平台、计费分组、服务档位和模型搜索按参考图的密集筛选带组织，并保留悦享 API 现有导航、字体、色彩与间距系统。
- [x] 桌面主表将“实付价格”和“基础价格”横向对照，突出绿色实付区，同时展示计费倍率。
- [x] 输入与输出明确区分 `<=272K` 和 `272K-1M`，缓存展示写入、读取与长上下文读取；基础价格由后端计费源直接返回，前端不反推。
- [x] Plus/Pro、Standard/Fast 和搜索交互已实测；Pro + Fast + `luna` 状态只展示目标模型且价格同步切换。
- [x] 390px 手机端改为纵向价格卡片，上下半屏均无横向溢出、文字截断、遮挡或价格跳位。
- [x] 页面控制台无应用报错；仅观察到用户浏览器扩展警告和本地域名 Turnstile 警告，与模型广场实现无关。

## 差异与取舍

- 参考图包含多家平台和大量分组；当前业务只展示用户实际可用的 OpenAI、Plus 和 Pro，不制造无效筛选项。
- 参考图为单独倍率筛选行；当前两个分组按钮直接携带有效倍率，减少重复控件并保留同样的价格解释能力。
- 参考图的 `0.299x` 是其平台价格；实现必须显示悦享 API 当前用户的真实 Plus/Pro 计费倍率和 BillingService 价格。

## 问题分级

- P0：无。
- P1：无。
- P2：无。

---

# Promotional Recharge Packages Design QA

QA date: 2026-07-24

## Inputs

- Reference: `C:\Users\ADMINI~1\AppData\Local\Temp\codex-clipboard-995a9022-f1d8-437f-9ada-04c5dac0d1c9.png`
- Production: `https://api-yue88.xyz/purchase`
- Desktop state: dark theme, collapsed sidebar, recharge tab, 36 CNY package selected
- Desktop viewport: reference 1056x624; production requested at 1024x624 and captured at the browser content size
- Mobile viewport: 390 CSS px, full-page production capture

## Artifacts

- Desktop production: `E:\中转\.tmp\design-qa\production-desktop-collapsed-full.png`
- Mobile production: `E:\中转\.tmp\design-qa\production-mobile-390-full.png`
- Full comparison: `E:\中转\.tmp\design-qa\comparison-full.png`
- Focused comparison: `E:\中转\.tmp\design-qa\comparison-focused.png`

## Comparison History

1. Compared the reference and production desktop captures together at the same selected-package state.
2. Repeated the comparison with focused crops covering the package cards, quick amounts, and custom amount input.
3. Inspected the full mobile payment flow for wrapping, clipping, overlap, and payment-summary consistency.

## Findings

- P0: none.
- P1: none.
- P2: none.
- The production layout keeps the existing product navigation, colors, spacing, card borders, and responsive grid.
- The four annotated reference prices were intentionally changed from 35/65/95/155 CNY to the requested 36/66/96/156 CNY; 300 CNY is unchanged.
- All five cards fit on one desktop row. The mobile layout uses two columns and leaves the fifth card on the final row without text overflow.
- Selecting each package updates the payment amount and credited USD balance consistently. Only WeChat Pay is visible.

result: passed

---

# Four-Tier Recharge Package Design QA

QA date: 2026-07-25

## Scope

- Replace the five promotional recharge cards with four cards while preserving the current payment page styling.
- Mark the second package as `推荐` and the fourth package as `大额推荐`.
- Keep the server-provided package list as the only source of recharge amounts.

## Verified States

- Desktop at 1440 x 900: four equal-width cards render in one row with the existing spacing, border, radius, typography, and color treatment.
- Mobile at 390 x 844: cards render as a 2 x 2 grid; labels, amounts, badges, and balance copy remain within their cards without overlap.
- Selection: the second package enters the selected visual state and exposes `aria-pressed="true"` after activation.
- Content: the cards display `38 -> 1000`, `72 -> 2000`, `105 -> 3000`, and `170 -> 5000`; the second and fourth badges display the requested Chinese copy.
- Source comparison: the implementation was compared in one view against the existing five-card payment page reference and retains its visual system while reducing the card count.
- Production: `release-v0.1.156.19` shows the same four cards on desktop and at 390 px; selecting the second card displays payment `72 CNY` and credited balance `2000 USD` without creating an order.

final result: passed

---

# 模型广场设计验收

验收日期：2026-08-02

final result: passed

## 对照范围

- 参考页面：`https://ohlao.cfd/monitor`
- 参考截图：`C:\Users\ADMINI~1\AppData\Local\Temp\codex-clipboard-b420c8b5-a8bc-4558-a5a0-0ce61bc0c48f.png`
- 生产页面：`https://api-yue88.xyz/model-plaza`
- 桌面视口：生产浏览器默认桌面视口
- 移动视口：390 x 844
- 并排对照：`C:\Users\Administrator\.codex\visualizations\2026\07\31\019fb763-2272-78d3-ba2d-33acd3eb4ac5\model-plaza\comparison.png`

## 视觉与交互结论

- [x] 顶栏和用户侧栏均提供“模型广场”入口，位置与现有导航层级一致。
- [x] 页面沿用悦享 API 现有浅色后台、品牌蓝、紧凑间距、表格和移动卡片规范，没有照搬第三方品牌视觉。
- [x] 六个收费模型完整展示输入、输出、缓存写入和缓存读取价格；价格单位明确为 USD / 1M Token。
- [x] Plus/Pro 分组、Standard/Fast 服务档位、模型搜索和手动刷新均完成真实交互验证。
- [x] 390px 手机端无横向溢出、文字截断或控件重叠，六个模型卡片和页尾说明均完整可读。
- [x] 页面价格直接来自 BillingService 当前生效计费规则，并包含用户分组倍率；不暴露上游账号、渠道和供应商成本。

## 差异说明

- OHLAO 参考页主要展示渠道健康状态；本次目标是让用户查看平台收费，因此主体使用更适合价格横向比较的桌面表格和移动卡片。
- 保留参考页顶栏快捷入口的发现方式，同时遵循悦享 API 已有导航与组件体系。

## 问题分级

- P0：无。
- P1：无。
- P2：无。
