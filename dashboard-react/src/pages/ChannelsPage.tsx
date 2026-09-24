import { useEffect, useState, type FormEvent } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createChannel, createChannelModel, deleteChannel, deleteChannelModel,
  listChannelHealth, listChannelModels, listChannels, loadRemoteModels,
  testChannel, updateChannel, updateChannelBalance, updateChannelModel,
  updateChannelStatus,
} from '../api/catalog'
import { AsyncState } from '../components/feedback/AsyncState'
import { Modal } from '../components/feedback/Modal'
import type { Channel, ChannelBalanceInput, ChannelInput, ChannelModel, ChannelModelInput, ChannelModelUpdate, ChannelTestItem, Health, RemoteModel } from '../types/api'

type Editor = Partial<Channel> | null

export function ChannelsPage() {
  const client = useQueryClient()
  const channels = useQuery({ queryKey: ['channels'], queryFn: listChannels, staleTime: 60_000 })
  const health = useQuery({ queryKey: ['channel-health'], queryFn: listChannelHealth, staleTime: 30_000 })
  const [editor, setEditor] = useState<Editor | undefined>()
  const [selected, setSelected] = useState<Channel | null>(null)
  const [error, setError] = useState('')
  const rows = channels.data?.list || []
  const healthMap = new Map((health.data?.list || []).map(row => [row.channel_id, row]))
  const refresh = async () => { await Promise.all([channels.refetch(), health.refetch()]) }
  const action = async (fn: () => Promise<unknown>) => { try { setError(''); await fn(); await refresh() } catch (reason) { setError(reason instanceof Error ? reason.message : '操作失败') } }

  return <>
    <AsyncState loading={channels.isLoading || health.isLoading} error={channels.error || health.error} hasData={Boolean(channels.data || health.data)} onRetry={() => { void refresh() }} />
    <section className="panel"><div className="panel-head"><div><h3>渠道管理</h3><p className="muted">上游渠道 · 状态 / 权重 / 优先级 / 余额</p></div><div className="panel-actions"><button className="button ghost" onClick={() => void refresh()}>刷新</button><button className="button primary" onClick={() => setEditor(null)}>新建渠道</button></div></div>{error && <p className="error">{error}</p>}<div className="table-wrap"><table><thead><tr><th>ID</th><th>名称</th><th>Base URL</th><th>状态</th><th>权重/优先级</th><th>余额</th><th>模型</th><th>熔断</th><th>操作</th></tr></thead><tbody>{rows.map(channel => { const state = healthMap.get(channel.id)?.state || 'closed'; return <tr key={channel.id}><td className="mono">#{channel.id}</td><td><b>{channel.name}</b></td><td className="mono channel-url">{channel.base_url}</td><td>{channel.status === 1 ? '启用' : '停用'}</td><td className="mono">{channel.weight} / {channel.priority}</td><td className="mono">{channel.balance == null ? '不限' : `$${channel.balance}`}</td><td>{channel.model_count}</td><td>{state}</td><td className="table-actions"><button className="button ghost" onClick={() => setEditor(channel)}>编辑</button><button className="button ghost" onClick={() => setSelected(channel)}>映射</button><button className="button ghost" onClick={() => void action(() => updateChannelStatus(channel.id, { status: channel.status === 1 ? 0 : 1 }))}>切换状态</button><button className="button delete" onClick={() => { if (window.confirm(`确认删除渠道「${channel.name}」？`)) void action(() => deleteChannel(channel.id)) }}>删除</button></td></tr> })}</tbody></table>{!rows.length && <div className="empty">暂无渠道</div>}</div></section>
    {editor !== undefined && <ChannelEditor channel={editor} onClose={() => setEditor(undefined)} onSaved={() => { setEditor(undefined); void refresh() }} />}
    {selected && <Mappings channel={selected} onClose={() => setSelected(null)} onChanged={() => void client.invalidateQueries({ queryKey: ['channels'] })} />}
  </>
}

function ChannelEditor({ channel, onClose, onSaved }: { channel: Editor; onClose: () => void; onSaved: () => void }) {
  return <Modal title={channel?.id ? `编辑渠道 · ${channel.name}` : '新建渠道'} submitText={channel?.id ? '保存' : '创建'} onClose={onClose} onSubmit={async event => { const form = new FormData(event.currentTarget); const apiKey = String(form.get('api_key') || ''); const input: Partial<ChannelInput> = { name: String(form.get('name') || ''), base_url: String(form.get('base_url') || ''), auth_type: String(form.get('auth_type') || 'bearer'), priority: Number(form.get('priority')), weight: Number(form.get('weight')), status: Number(form.get('status')) }; if (apiKey) input.api_key = apiKey; if (form.get('balance')) input.balance = String(form.get('balance')); if (!channel?.id && !apiKey) throw new Error('新建渠道必须填写上游 API Key'); if (channel?.id) await updateChannel(channel.id, input); else await createChannel(input as ChannelInput); onSaved() }}><div className="form-grid"><Field label="渠道名称" name="name" defaultValue={channel?.name} required /><Field label="Base URL" name="base_url" defaultValue={channel?.base_url} required /><Field label="上游 API Key" name="api_key" required={!channel?.id} /><Field label="优先级" name="priority" type="number" defaultValue={channel?.priority ?? 0} /><Field label="权重" name="weight" type="number" defaultValue={channel?.weight ?? 100} /><Field label="余额" name="balance" defaultValue={channel?.balance || ''} /><label><span>状态</span><select name="status" defaultValue={String(channel?.status ?? 1)}><option value="1">启用</option><option value="0">停用</option></select></label></div></Modal>
}

function Mappings({ channel, onClose, onChanged }: { channel: Channel; onClose: () => void; onChanged: () => void }) {
  const query = useQuery({ queryKey: ['channel-models', channel.id], queryFn: () => listChannelModels(channel.id) })
  const [editor, setEditor] = useState<ChannelModel | null | undefined>()
  const [remote, setRemote] = useState<RemoteModel[]>([])
  const loadRemote = async () => { const result = await loadRemoteModels(channel.id); if (!result.ok) throw new Error(result.error || '拉取失败'); setRemote(result.models || []) }
  const save = async (input: ChannelModelInput | ChannelModelUpdate) => { if (editor) await updateChannelModel(channel.id, editor.id, input as ChannelModelUpdate); else await createChannelModel(channel.id, input as ChannelModelInput); setEditor(undefined); onChanged(); await query.refetch() }
  return <Modal title={`模型映射 · ${channel.name}`} onClose={onClose}><div className="modal-toolbar"><button type="button" className="button ghost" onClick={() => void loadRemote()}>拉取远端模型</button><button type="button" className="button primary" onClick={() => setEditor(null)}>手动添加映射</button></div>{remote.length > 0 && <p className="muted">远端模型：{remote.map(item => item.id).join(', ')}</p>}{query.isLoading ? <div className="empty">加载中…</div> : <div className="table-wrap"><table><thead><tr><th>对外模型</th><th>上游模型</th><th>状态</th><th /></tr></thead><tbody>{query.data?.list.map(row => <tr key={row.id}><td>{row.model_name}</td><td>{row.upstream_model}</td><td>{row.enabled ? '启用' : '停用'}</td><td><button className="button ghost" onClick={() => setEditor(row)}>编辑</button><button className="button delete" onClick={() => { if (window.confirm('删除该映射？')) void deleteChannelModel(channel.id, row.id).then(() => { onChanged(); void query.refetch() }) }}>删除</button></td></tr>)}</tbody></table></div>}{editor !== undefined && <MappingEditor mapping={editor} onClose={() => setEditor(undefined)} onSave={save} />}</Modal>
}

function MappingEditor({ mapping, onClose, onSave }: { mapping: ChannelModel | null; onClose: () => void; onSave: (input: ChannelModelInput | ChannelModelUpdate) => Promise<void> }) { return <Modal title={mapping ? '编辑映射' : '添加映射'} submitText="保存" onClose={onClose} onSubmit={async event => { const form = new FormData(event.currentTarget); await onSave({ model_name: String(form.get('model_name') || ''), upstream_model: String(form.get('upstream_model') || ''), enabled: form.get('enabled') === 'on' }) }}><Field label="对外模型名" name="model_name" defaultValue={mapping?.model_name} readOnly={Boolean(mapping)} required /><Field label="上游模型名" name="upstream_model" defaultValue={mapping?.upstream_model} required /><label className="checkbox"><input name="enabled" type="checkbox" defaultChecked={mapping?.enabled ?? true} /> 启用</label></Modal> }
function Field({ label, ...props }: React.InputHTMLAttributes<HTMLInputElement> & { label: string }) { return <label><span>{label}</span><input {...props} /></label> }
