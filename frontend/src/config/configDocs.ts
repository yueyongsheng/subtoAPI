export const CONFIG_DOCS_BASE_URL = 'https://api-yue88.xyz/v1'
export const CONFIG_DOCS_API_KEY = 'YOUR_YUEXIANG_API_KEY'
export const CONFIG_DOCS_MODEL = 'gpt-5.6-sol'
export const CONFIG_DOCS_RESPONSES_URL = `${CONFIG_DOCS_BASE_URL}/responses`

export type ConfigDocIcon = 'terminal' | 'cog' | 'cpu' | 'swap' | 'beaker' | 'book'

export interface ConfigDocCodeBlock {
  label: string
  language: string
  code: string
}

export interface ConfigDocSection {
  id: string
  title: string
  description?: string
  steps?: string[]
  codeBlocks?: ConfigDocCodeBlock[]
  note?: string
}

export interface ConfigGuide {
  id: string
  badge: string
  minutes: number
  title: string
  summary: string
  icon: ConfigDocIcon
  sections: ConfigDocSection[]
}

const codexConfig = `model_provider = "yuexiang"
model = "${CONFIG_DOCS_MODEL}"
model_reasoning_effort = "high"
preferred_auth_method = "apikey"

[model_providers.yuexiang]
name = "悦享 API"
base_url = "${CONFIG_DOCS_BASE_URL}"
wire_api = "responses"
requires_openai_auth = true`

const authJson = `{
  "OPENAI_API_KEY": "${CONFIG_DOCS_API_KEY}"
}`

const curlCommand = `curl ${CONFIG_DOCS_RESPONSES_URL} \\
  -H "Authorization: Bearer ${CONFIG_DOCS_API_KEY}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"${CONFIG_DOCS_MODEL}","input":"你好，请回复：连接成功"}'`

const powershellRequest = `$headers = @{
  Authorization = "Bearer ${CONFIG_DOCS_API_KEY}"
  "Content-Type" = "application/json"
}
$body = @{
  model = "${CONFIG_DOCS_MODEL}"
  input = "你好，请回复：连接成功"
} | ConvertTo-Json

Invoke-RestMethod -Method Post \
  -Uri "${CONFIG_DOCS_RESPONSES_URL}" \
  -Headers $headers -Body $body`

export const configGuides: ConfigGuide[] = [
  {
    id: 'codex-app',
    badge: '桌面应用',
    minutes: 8,
    title: 'Codex App 配置',
    summary: '在 Codex App 中添加悦享 API 自定义模型提供商，并用 API Key 登录。',
    icon: 'cog',
    sections: [
      {
        id: 'prepare',
        title: '1. 准备 API Key',
        description: '登录悦享 API 控制台，在 API Keys 页面创建密钥。密钥只会完整显示一次，请妥善保存。',
        steps: ['不要使用管理员登录凭据代替 API Key。', '建议为每台设备单独创建密钥，便于停用和排查。'],
      },
      {
        id: 'config',
        title: '2. 写入 Codex 配置',
        description: '打开用户目录下的 .codex/config.toml，将下面的提供商配置写入文件。',
        codeBlocks: [{ label: '~/.codex/config.toml', language: 'toml', code: codexConfig }],
      },
      {
        id: 'auth',
        title: '3. 写入 API Key',
        description: '在同一目录创建 auth.json。请将占位符替换为控制台生成的真实 API Key。',
        codeBlocks: [{ label: '~/.codex/auth.json', language: 'json', code: authJson }],
        note: '不要把 auth.json 上传到 Git、网盘公开目录或聊天群。',
      },
      {
        id: 'verify',
        title: '4. 重启并验证',
        description: '完全退出 Codex App 后重新打开，发起一个简短请求。若模型正常回复，即配置完成。',
        steps: ['模型应显示为 gpt-5.6-sol。', '请求协议应使用 Responses API。', '首次测试建议只发送短文本，便于定位配置问题。'],
      },
    ],
  },
  {
    id: 'codex-cli-windows',
    badge: 'Windows',
    minutes: 10,
    title: 'Codex CLI Windows',
    summary: '适用于 Windows 10/11，完成 Codex CLI 安装、配置和 PowerShell 验证。',
    icon: 'terminal',
    sections: [
      {
        id: 'install',
        title: '1. 安装 Codex CLI',
        description: '先安装 Node.js LTS，然后在 PowerShell 中全局安装 Codex CLI。',
        codeBlocks: [{ label: 'PowerShell', language: 'powershell', code: 'npm install -g @openai/codex\ncodex --version' }],
      },
      {
        id: 'files',
        title: '2. 创建配置文件',
        description: '创建 %USERPROFILE%\\.codex 目录，并分别写入 config.toml 与 auth.json。',
        codeBlocks: [
          { label: 'PowerShell', language: 'powershell', code: 'New-Item -ItemType Directory -Force "$env:USERPROFILE\\.codex"' },
          { label: 'config.toml', language: 'toml', code: codexConfig },
          { label: 'auth.json', language: 'json', code: authJson },
        ],
      },
      {
        id: 'run',
        title: '3. 启动 Codex',
        description: '重新打开 PowerShell，在项目目录执行 Codex。',
        codeBlocks: [{ label: 'PowerShell', language: 'powershell', code: 'cd C:\\path\\to\\your-project\ncodex' }],
      },
      {
        id: 'verify',
        title: '4. 独立验证网关',
        description: '若 CLI 报错，可先绕过客户端直接验证 Base URL、API Key 和模型名。',
        codeBlocks: [{ label: 'PowerShell', language: 'powershell', code: powershellRequest }],
      },
    ],
  },
  {
    id: 'codex-cli-macos-linux',
    badge: 'Terminal',
    minutes: 9,
    title: 'Codex CLI macOS / Linux',
    summary: '适用于 macOS、Ubuntu、Debian、CentOS 等终端环境。',
    icon: 'terminal',
    sections: [
      {
        id: 'install',
        title: '1. 安装 Codex CLI',
        description: '确认 Node.js 和 npm 可用后安装 Codex CLI。',
        codeBlocks: [{ label: 'Terminal', language: 'bash', code: 'npm install -g @openai/codex\ncodex --version' }],
      },
      {
        id: 'config',
        title: '2. 创建配置目录',
        description: '在用户主目录创建 .codex，并写入提供商配置。',
        codeBlocks: [
          { label: 'Terminal', language: 'bash', code: 'mkdir -p ~/.codex\nchmod 700 ~/.codex' },
          { label: '~/.codex/config.toml', language: 'toml', code: codexConfig },
          { label: '~/.codex/auth.json', language: 'json', code: authJson },
        ],
      },
      {
        id: 'permissions',
        title: '3. 收紧密钥文件权限',
        codeBlocks: [{ label: 'Terminal', language: 'bash', code: 'chmod 600 ~/.codex/auth.json' }],
      },
      {
        id: 'verify',
        title: '4. 启动与验证',
        description: '进入项目目录执行 codex；如需独立检查网关，可运行 curl 测试。',
        codeBlocks: [
          { label: 'Terminal', language: 'bash', code: 'cd /path/to/your-project\ncodex' },
          { label: 'curl', language: 'bash', code: curlCommand },
        ],
      },
    ],
  },
  {
    id: 'ide',
    badge: 'IDE',
    minutes: 8,
    title: 'VS Code / Cursor / JetBrains',
    summary: '为支持 OpenAI Compatible Provider 的 IDE 插件填写统一连接参数。',
    icon: 'cpu',
    sections: [
      {
        id: 'provider',
        title: '1. 选择兼容提供商',
        description: '在插件设置中选择 OpenAI Compatible、Custom OpenAI 或同等选项。不要选择仅支持官方账号登录的模式。',
      },
      {
        id: 'parameters',
        title: '2. 填写连接参数',
        steps: [
          `Base URL：${CONFIG_DOCS_BASE_URL}`,
          `API Key：${CONFIG_DOCS_API_KEY}`,
          `Model：${CONFIG_DOCS_MODEL}`,
          'API 类型：Responses API（若插件仅支持 Chat Completions，请先确认兼容性）',
        ],
      },
      {
        id: 'verify',
        title: '3. 保存并测试',
        description: '使用插件内置的 Test、Check Connection 或发送短消息功能验证。若插件自动在 Base URL 后追加 /v1，请改填 https://api-yue88.xyz。',
      },
    ],
  },
  {
    id: 'cc-switch',
    badge: '切换器',
    minutes: 7,
    title: 'CC Switch 配置',
    summary: '在 CC Switch 中新增悦享 API，并作为 Codex 的 OpenAI 兼容提供商使用。',
    icon: 'swap',
    sections: [
      {
        id: 'add',
        title: '1. 新增供应商',
        description: '打开 CC Switch 的供应商管理，选择 Codex 或 OpenAI Compatible 类型并新建配置。',
      },
      {
        id: 'fields',
        title: '2. 填写字段',
        steps: [
          '名称：悦享 API',
          `请求地址：${CONFIG_DOCS_BASE_URL}`,
          `API Key：${CONFIG_DOCS_API_KEY}`,
          `默认模型：${CONFIG_DOCS_MODEL}`,
        ],
      },
      {
        id: 'activate',
        title: '3. 启用并验证',
        description: '保存后切换到“悦享 API”，重新打开 Codex 客户端并发送短消息。旧进程可能仍读取切换前的环境变量，需要完全退出后再启动。',
      },
    ],
  },
  {
    id: 'sdk-curl',
    badge: 'API',
    minutes: 6,
    title: 'SDK 与 curl 调用',
    summary: '直接使用 Responses API 接入服务，适合脚本、后端服务和连通性排查。',
    icon: 'beaker',
    sections: [
      {
        id: 'curl',
        title: '1. curl 请求',
        description: '将示例中的 API Key 占位符替换为自己的密钥。',
        codeBlocks: [{ label: 'Terminal', language: 'bash', code: curlCommand }],
      },
      {
        id: 'javascript',
        title: '2. JavaScript SDK',
        description: '使用 OpenAI SDK 时，将 baseURL 指向悦享 API。',
        codeBlocks: [{
          label: 'JavaScript',
          language: 'javascript',
          code: `import OpenAI from 'openai'\n\nconst client = new OpenAI({\n  apiKey: '${CONFIG_DOCS_API_KEY}',\n  baseURL: '${CONFIG_DOCS_BASE_URL}',\n})\n\nconst response = await client.responses.create({\n  model: '${CONFIG_DOCS_MODEL}',\n  input: '你好，请回复：连接成功',\n})\n\nconsole.log(response.output_text)`,
        }],
      },
      {
        id: 'python',
        title: '3. Python SDK',
        codeBlocks: [{
          label: 'Python',
          language: 'python',
          code: `from openai import OpenAI\n\nclient = OpenAI(\n    api_key="${CONFIG_DOCS_API_KEY}",\n    base_url="${CONFIG_DOCS_BASE_URL}",\n)\n\nresponse = client.responses.create(\n    model="${CONFIG_DOCS_MODEL}",\n    input="你好，请回复：连接成功",\n)\n\nprint(response.output_text)`,
        }],
      },
    ],
  },
]

export const configDocFaqs = [
  {
    question: 'Base URL 应该填到哪里？',
    answer: `通常填写 ${CONFIG_DOCS_BASE_URL}。如果客户端会自动追加 /v1，则填写 https://api-yue88.xyz。`,
  },
  {
    question: '为什么返回 401？',
    answer: '优先检查 API Key 是否完整、是否已停用，以及 Authorization 是否使用 Bearer 格式。',
  },
  {
    question: '为什么返回模型不存在？',
    answer: `确认模型名填写为 ${CONFIG_DOCS_MODEL}，并确认当前 API Key 有权访问对应分组。`,
  },
  {
    question: 'API Key 可以发给客服排查吗？',
    answer: '请勿发送完整密钥。排查时只提供请求时间、错误信息和密钥末四位。',
  },
]

export function getConfigGuide(id: string): ConfigGuide | undefined {
  return configGuides.find((guide) => guide.id === id)
}
