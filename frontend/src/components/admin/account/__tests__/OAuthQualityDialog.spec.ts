import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import OAuthQualityDialog from '../OAuthQualityDialog.vue'
import { listOAuthQualityAccounts, runOAuthQuality, type OAuthQualityAccount } from '@/api/admin/oauthQuality'
vi.mock('@/api/admin/oauthQuality', async original => ({ ...await original<typeof import('@/api/admin/oauthQuality')>(), listOAuthQualityAccounts: vi.fn(), runOAuthQuality: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: {} }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, args?: object) => key + (args ? JSON.stringify(args) : '') }) }))
const fixtures: OAuthQualityAccount[] = [
 { id:1, name:'OAuth one', platform:'openai', type:'oauth', group_ids:[7], status:'active', schedulable:true },
 { id:2, name:'Key two', platform:'openai', type:'apikey', group_ids:[7], status:'active', schedulable:true },
 { id:3, name:'Other', platform:'openai', type:'oauth', group_ids:[8], status:'active', schedulable:true }
]
const render = (extra = {}) => mount(OAuthQualityDialog, {
 props: { show:true, groupId:7, scopeLabel:'group-seven', groups:[{id:7,name:'group-seven'},{id:8,name:'group-eight'}], ...extra },
 global: { stubs: { BaseDialog: { template:'<div><slot/><slot name="footer"/></div>' } } }
})
describe('Account quality selection', () => {
 beforeEach(() => {
  vi.mocked(listOAuthQualityAccounts).mockReset().mockImplementation(async (group,types) => fixtures.filter(a => (group == null || a.group_ids.includes(group)) && types.includes(a.type)))
  vi.mocked(runOAuthQuality).mockReset().mockImplementation(async (group,ids,model,types,keys,custom) => ({
   checked_at:new Date().toISOString(), group_id:group, model_id:model, account_types:types, probe_keys:keys, custom_probe:custom,
   accounts:ids.map(id => ({ ...fixtures.find(a => a.id === id)!, status:'passed' as const, summary:'matched', passed:keys.length, total:keys.length,
    probes:keys.map(key => ({key,label:key,status:'passed' as const,summary:'matched',output_preview:'demo'})) }))
  }))
 })
 it('inherits main-table selection and executes only the chosen method for an API key', async () => {
  const w=render({initialAccountIds:[2],initialTypes:['apikey']})
  await flushPromises()
  expect(w.get('[data-testid="account-2"]').element).toHaveProperty('checked',true)
  expect(w.find('[data-testid="account-1"]').exists()).toBe(false)
  await w.get('[data-testid="probe-svg_html"]').setValue(false)
  await w.get('[data-testid="probe-english_knowledge"]').setValue(true)
  await w.get('[data-testid="run-quality"]').trigger('click')
  await flushPromises()
  expect(runOAuthQuality).toHaveBeenCalledWith(7,[2],'gpt-6-astra',['apikey'],['english_knowledge'],undefined)
  expect(w.text()).toContain('oauthQuality.completed')
  expect(w.findAll('tbody tr')).toHaveLength(1)
  w.unmount()
 })
 it('loads all matching types, selects a batch, and sends only selected questions', async () => {
  const w=render()
  await flushPromises()
  expect(listOAuthQualityAccounts).toHaveBeenCalledWith(7,['oauth','apikey'])
  await w.get('[data-testid="select-visible"]').trigger('click')
  await w.get('[data-testid="probe-reasoning_exact"]').setValue(true)
  await w.get('[data-testid="run-quality"]').trigger('click')
  await flushPromises()
  expect(runOAuthQuality).toHaveBeenCalledTimes(4)
  expect(vi.mocked(runOAuthQuality).mock.calls.map(call=>call[1])).toEqual([[1],[1],[2],[2]])
  expect(vi.mocked(runOAuthQuality).mock.calls.map(call=>call[4])).toEqual([['svg_html'],['reasoning_exact'],['svg_html'],['reasoning_exact']])
  expect(w.find('iframe').exists()).toBe(true)
  expect(w.find('iframe').attributes('sandbox')).toBe('allow-scripts')
  w.unmount()
 })
 it('clears selection on scope changes and discards late results', async () => {
  let resolveOld!: (accounts:OAuthQualityAccount[])=>void
  vi.mocked(listOAuthQualityAccounts).mockImplementationOnce(()=>new Promise(resolve=>{resolveOld=resolve}))
  const w=render({initialAccountIds:[1]})
  await flushPromises()
  await w.get('[data-testid="quality-group"]').setValue('8')
  await flushPromises()
  resolveOld(fixtures.slice(0,2))
  await flushPromises()
  expect(w.find('[data-testid="account-1"]').exists()).toBe(false)
  expect(w.find('[data-testid="account-3"]').exists()).toBe(true)
  expect(w.get('[data-testid="run-quality"]').attributes('disabled')).toBeDefined()
  w.unmount()
 })
 it('requires a selected method and nonempty custom question, then transmits optional expected text', async () => {
  const w=render({initialAccountIds:[1]})
  await flushPromises()
  await w.get('[data-testid="probe-svg_html"]').setValue(false)
  expect(w.get('[data-testid="run-quality"]').attributes('disabled')).toBeDefined()
  await w.get('[data-testid="probe-custom"]').setValue(true)
  expect(w.get('[data-testid="run-quality"]').attributes('disabled')).toBeDefined()
  await w.get('[data-testid="custom-prompt"]').setValue('How many days in a week?')
  await w.get('[data-testid="custom-expected"]').setValue('7')
  await w.get('[data-testid="run-quality"]').trigger('click')
  await flushPromises()
  expect(runOAuthQuality).toHaveBeenCalledWith(7,[1],'gpt-6-astra',['oauth','apikey'],['custom'],{prompt:'How many days in a week?',expected:'7'})
  w.unmount()
 })
 it('reports hidden selections during search and never silently truncates all-selection to fifty', async () => {
  vi.mocked(listOAuthQualityAccounts).mockResolvedValue(Array.from({length:55},(_,i)=>({...fixtures[0],id:i+10})))
  const w=render()
  await flushPromises()
  await w.get('[data-testid="select-visible"]').trigger('click')
  expect(w.text()).toContain('"accounts":55')
  await w.get('[data-testid="quality-search"]').setValue('#10')
  expect(w.text()).toContain('"count":54')
  await w.get('[data-testid="type-apikey"]').setValue(false)
  await flushPromises()
  expect(w.get('[data-testid="run-quality"]').attributes('disabled')).toBeDefined()
  w.unmount()
 })
 it('loads preview types and custom question locally without invoking APIs', async () => {
  const w=render({preview:true,groupId:2,initialTypes:['apikey'],initialAccountIds:[201]})
  await flushPromises()
  expect(w.find('[data-testid="account-201"]').exists()).toBe(true)
  expect(w.find('[data-testid="account-186"]').exists()).toBe(false)
  expect(listOAuthQualityAccounts).not.toHaveBeenCalled()
  expect(runOAuthQuality).not.toHaveBeenCalled()
  w.unmount()
 })
 it('stops subsequent accounts after a rejected request and preserves finished results', async () => {
  const w=render()
  await flushPromises()
  vi.mocked(runOAuthQuality).mockRejectedValueOnce(new Error('network'))
  await w.get('[data-testid="select-visible"]').trigger('click')
  await w.get('[data-testid="run-quality"]').trigger('click')
  await flushPromises()
  expect(runOAuthQuality).toHaveBeenCalledTimes(1)
  expect(w.find('[role="alert"]').text()).toContain('oauthQuality.runFailed')
  w.unmount()
 })
})

