import type { OAuthQualityAccount, OAuthQualityAccountResult, OAuthQualityProbeResult, QualityCustomProbe, QualityProbeKey } from '@/api/admin/oauthQuality'

export const previewQualityAccounts: OAuthQualityAccount[] = [
  { id: 71, name: 'OpenAI Plus · Astra', platform: 'openai', type: 'oauth', group_ids: [2], status: 'active', schedulable: true },
  { id: 184, name: 'OpenAI Pro · Astra', platform: 'openai', type: 'oauth', group_ids: [3], status: 'active', schedulable: true },
  { id: 186, name: 'OpenAI Plus · Sol', platform: 'openai', type: 'oauth', group_ids: [2], status: 'active', schedulable: true },
  { id: 197, name: '备用 OAuth（额度等待）', platform: 'openai', type: 'oauth', group_ids: [2], status: 'active', schedulable: false },
  { id: 201, name: 'OpenAI Plus · API Key', platform: 'openai', type: 'apikey', group_ids: [2], status: 'active', schedulable: true },
  { id: 202, name: 'OpenAI Pro · API Key', platform: 'openai', type: 'apikey', group_ids: [3], status: 'active', schedulable: true },
  { id: 203, name: 'Grok · API Key', platform: 'grok', type: 'apikey', group_ids: [7], status: 'active', schedulable: true },
  { id: 204, name: '未分组 · API Key', platform: 'openai', type: 'apikey', group_ids: [], status: 'active', schedulable: true },
  { id: 205, name: 'Claude · Setup Token', platform: 'anthropic', type: 'setup-token', group_ids: [9], status: 'active', schedulable: true },
  { id: 206, name: 'Claude · AWS Bedrock', platform: 'anthropic', type: 'bedrock', group_ids: [9], status: 'active', schedulable: true },
]

export function previewQualityResult(account: OAuthQualityAccount, keys: QualityProbeKey[], custom?: QualityCustomProbe): OAuthQualityAccountResult {
  const probes = keys.map((key): OAuthQualityProbeResult => {
    const base = { key, label: key, latency_ms: 1180 + account.id }
    if (account.id === 197) return { ...base, status: 'failed', summary: '演示：上游请求失败，未获得有效回答。', output_preview: '' }
    switch (key) {
      case 'svg_html': return { ...base, status: 'review', summary: '演示：SVG 动画结构已返回，画面质量待人工复核。', output_preview: '<html><body><svg viewBox="0 0 320 180"><text x="20" y="70">pelican / bicycle</text><circle cx="80" cy="130" r="24"/><circle cx="240" cy="130" r="24"/><animate attributeName="opacity" values="1;0.6;1" dur="2s" repeatCount="indefinite"/></svg></body></html>' }
      case 'reasoning_exact': return { ...base, status: account.id === 201 ? 'review' : 'passed', summary: account.id === 201 ? '演示：答案与预期 21 不一致。' : '演示：最终答案 21 正确。', output_preview: account.id === 201 ? '29' : '21' }
      case 'english_knowledge': return { ...base, status: 'passed', summary: '演示：与参考回答 yes 一致，单题不构成能力鉴定。', output_preview: 'yes' }
      case 'structured_output': return { ...base, status: 'passed', summary: '演示：JSON 字段和类型符合预期。', output_preview: '{"answer":42,"ok":true}' }
      case 'custom': return { ...base, status: 'review', summary: '演示：正式检测会提交自定义问题并按预期答案匹配；此处仅展示布局。', output_preview: '演示回答（未调用模型）\n问题：' + (custom?.prompt ?? '') }
    }
  })
  const passed = probes.filter(p => p.status === 'passed').length
  return { ...account, status: probes.every(p => p.status === 'failed') ? 'failed' : passed === probes.length ? 'passed' : 'review', summary: '演示结果，仅展示所选方法。', passed, total: probes.length, probes }
}
