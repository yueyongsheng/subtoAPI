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
